import streamlit as st
import os
import git
import subprocess
import re
from streamlit_mermaid import st_mermaid
import shutil
import signal
import queue
import threading
import time
import random
import streamlit_shadcn_ui as ui
import logging



st.set_page_config(
    page_title="DORA",  # This is the browser tab title
    page_icon="",                # Optional: emoji or URL to favicon
    layout="centered"                  # Optional: 'centered' or 'wide'
)


# --- Logging Configuration ---
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')


# --- Session State Initialization ---
if 'repo_ports' not in st.session_state:
    st.session_state.repo_ports = {}
if 'running_servers' not in st.session_state:
    st.session_state.running_servers = {} # port -> process
if 'next_port' not in st.session_state:
    st.session_state.next_port = 8005
if 'selected_repo_index' not in st.session_state:
    st.session_state.selected_repo_index = 0
if 'selected_repo' not in st.session_state:
    st.session_state.selected_repo = None
if 'current_process' not in st.session_state:
    st.session_state.current_process = None
if 'last_analysis_status' not in st.session_state:
    st.session_state.last_analysis_status = None

# --- Product Use Case Session State Initialization ---
if 'selected_use_case_index' not in st.session_state:
    st.session_state.selected_use_case_index = 0
if 'selected_use_case' not in st.session_state:
    st.session_state.selected_use_case = None



def run_command_in_thread(process, q):
    try:
        for line in iter(process.stdout.readline, ''):
            # Log the output to the console
            line_without_newline = line.strip()
            if line_without_newline:
                logging.info(line_without_newline)
            q.put(line)
        process.stdout.close()
        return_code = process.wait()
        q.put(f"---RC:{return_code}---")
    except Exception as e:
        logging.error(f"Error in command thread: {e}")
        q.put(str(e))
        q.put("---RC:1---")


def start_docs_server(repo_path, port):
    # Kill any existing server on this specific port
    if port in st.session_state.running_servers:
        proc = st.session_state.running_servers[port]
        if proc.poll() is None:
            st.warning(f"Stopping previous documentation server (PID: {proc.pid}) on port {port}...")
            try:
                os.killpg(os.getpgid(proc.pid), signal.SIGKILL)
                proc.wait()
            except ProcessLookupError:
                pass  # Process already dead
            finally:
                if port in st.session_state.running_servers:
                    del st.session_state.running_servers[port]
    
    # Give the OS a moment to release the port
    time.sleep(2)

    ra_path = os.path.join(repo_path, '_ra')
    if not os.path.isdir(ra_path):
        st.error(f"Directory not found: {ra_path}. Cannot start documentation server.")
        return

    try:
        # Step 1: Install documentation dependencies
        with st.spinner("Installing documentation dependencies..."):
            install_process = subprocess.run(
                [
                    "pip", "install",
                    "mkdocs>=1.5.0",
                    "mkdocs-material>=9.0.0",
                    "mkdocs-mermaid2-plugin>=1.0.0",
                    "pymdown-extensions>=10.0.0",
                    "mkdocs-awesome-pages-plugin>=2.8.0",
                    "mkdocs-minify-plugin>=0.7.0",
                    "mkdocs-git-revision-date-localized-plugin>=1.2.0"
                ],
                cwd=ra_path,
                capture_output=True,
                text=True,
                check=False
            )
            if install_process.returncode != 0:
                st.error("Failed to install dependencies:")
                st.code(install_process.stderr)
                return

        # Step 2: Build docs
        with st.spinner("Building documentation..."):
            build_process = subprocess.run(
                ["mkdocs", "build", "--clean"],
                cwd=ra_path,
                capture_output=True,
                text=True,
                check=False
            )
            if build_process.returncode != 0:
                st.error("Failed to build documentation:")
                st.code(build_process.stderr)
                return
        
        # Step 3: Serve docs
        with st.spinner(f"Starting documentation server on port {port}..."):
            proc = subprocess.Popen(
                ["mkdocs", "serve", f"--dev-addr=127.0.0.1:{port}"],
                cwd=ra_path,
                preexec_fn=os.setsid
            )
            st.session_state.running_servers[port] = proc
            st.success("Documentation server started.")
            
            # Centered button to view docs
            _, col, _ = st.columns([1, 2, 1])
            with col:
                st.link_button("View Documentation", url=f"http://localhost:{port}", use_container_width=True)

    except Exception as e:
        st.error(f"An error occurred while setting up the documentation server: {e}")

@st.fragment
def show_analysis_progress():
    # Determine what entity is being analyzed (repo or use case)
    target_entity = st.session_state.get('target_entity_running')
    if not target_entity:
        # Fallback: infer from repo_path_running
        inferred = None
        try:
            running_path = st.session_state.get('repo_path_running')
            if running_path:
                base = os.path.basename(os.path.normpath(running_path))
                if os.path.isfile(os.path.join(running_path, 'usecase.md')):
                    inferred = f"use case: {base}"
                else:
                    inferred = f"repo: {base}"
        except Exception:
            inferred = None
        target_entity = inferred or st.session_state.get('selected_use_case') or st.session_state.get('selected_repo') or "(unknown)"

    with st.status(f"Running analysis: `{st.session_state.selected_command_name_running}` on `{target_entity}`...", expanded=True) as status:
        log_placeholder = st.empty()

        # Loop to update the log
        while st.session_state.get('command_is_running'):
            while not st.session_state.command_q.empty():
                line = st.session_state.command_q.get_nowait()
                if line.startswith("---RC:"):
                    st.session_state.command_return_code = int(line.replace("---RC:", "").replace("---", ""))
                    st.session_state.command_is_running = False
                    st.session_state.current_process = None
                    break
                st.session_state.command_log += line
            
            log_placeholder.code(st.session_state.command_log)
            
            if not st.session_state.get('command_is_running'):
                # Command has just finished
                if st.session_state.command_return_code == 0:
                    # Post-run actions: if it was a doc generation, start the server.
                    if "documentation" in st.session_state.selected_command_name_running.lower():
                        status.update(label="Documentation generated!", state="complete", expanded=False)
                        logging.info("Documentation generation successful.")
                        st.success("Documentation generation complete. Starting server...")
                        running_repo_path = st.session_state.repo_path_running
                        if running_repo_path not in st.session_state.repo_ports:
                            st.session_state.repo_ports[running_repo_path] = st.session_state.next_port
                            st.session_state.next_port += 1
                        port_to_start = st.session_state.repo_ports[running_repo_path]
                        start_docs_server(running_repo_path, port_to_start)
                    else:
                        status.update(label="Analysis complete!", state="complete", expanded=False)
                        logging.info(f"Analysis successful: {st.session_state.selected_command_name_running}")
                        st.session_state.last_analysis_status = {"status": "success", "message": "Analysis complete!"}

                else:
                    status.update(label="Analysis failed!", state="error")
                    logging.error(f"Analysis failed: {st.session_state.selected_command_name_running} with exit code {st.session_state.command_return_code}")
                    st.session_state.last_analysis_status = {"status": "error", "message": f"Analysis failed with exit code: {st.session_state.command_return_code}"}
                
                # Rerun one last time to clear the spinner
                st.session_state.current_process = None
                st.session_state.target_entity_running = None
                
            else:
                time.sleep(1) # The fragment will re-run itself, not the whole app

st.title("DORA")
st.caption("Documentation, Obsolescence, Risk and Architecture Assistant")


analysis_type = st.radio(
    "Choose analysis type:",
    ("Repo Analysis", "Product Use Case Analysis")
)

if analysis_type == "Repo Analysis":
    st.header("Repo Analysis")

    # Mirror Product Use Case Analysis: show selected entity caption before tabs
    if st.session_state.get('selected_repo'):
        st.caption(f"Selected Repository: `{st.session_state.get('selected_repo')}`")

    tab_options = ["Repository"]
    if st.session_state.get('selected_repo'):
        tab_options.extend(["Analysis", "Results"])
    # Determine default tab
    if st.session_state.get('command_is_running'):
        # Command is running, default to Analysis tab
        default_tab = "Analysis"
    else:
        # Default to Repository
        default_tab = "Repository"
else:
    st.header("Product Use Case Analysis")

    # Mirror Repo Analysis: show selected entity caption before tabs
    if st.session_state.get('selected_use_case'):
        st.caption(f"Selected Use Case: `{st.session_state.get('selected_use_case')}`")

    # Tabs for parity with Repo Analysis (only "Use Case" is functional now)
    uc_tab_options = ["Use Case", "Analysis", "Results"]
    # Default to Use Case tab
    uc_default_tab = "Use Case"
    selected_uc_tab = ui.tabs(options=uc_tab_options, default_value=uc_default_tab)

    if selected_uc_tab == "Use Case":
        with st.expander("Create a new Use Case"):
            use_case_name = st.text_input(
                "Use case name (identifier)",
                placeholder="e.g., checkout-service or product-search",
                key="uc_name_input"
            )
            use_case_desc = st.text_area(
                "Describe your product use case:",
                placeholder="Enter a paragraph describing your product or feature...",
                key="uc_desc_input"
            )

            def _slugify(name: str) -> str:
                slug = re.sub(r"[^a-zA-Z0-9._-]+", "-", name.strip()).strip("-").lower()
                return slug or f"usecase-{int(time.time())}"

            if st.button("Save Use Case", key="save_use_case_btn"):
                if not use_case_name.strip() or not use_case_desc.strip():
                    st.warning("Please provide both a use case name and description.")
                else:
                    slug = _slugify(use_case_name)
                    uc_dir = os.path.join("../workspace", slug)
                    os.makedirs(uc_dir, exist_ok=True)
                    md_path = os.path.join(uc_dir, "usecase.md")
                    with open(md_path, 'w', encoding='utf-8') as f:
                        f.write(f"# {use_case_name}\n\n{use_case_desc}\n")

                    # Mirror clone flow for use cases: set as selected and index
                    st.session_state.selected_use_case = slug
                    try:
                        entries = [d for d in os.listdir("../workspace") if os.path.isfile(os.path.join("../workspace", d, "usecase.md"))]
                        if slug in entries:
                            st.session_state.selected_use_case_index = entries.index(slug)
                    except Exception:
                        pass

                    st.success(f"Saved use case in: {uc_dir}")
                    st.rerun()

        # --- List Saved Use Cases and Select (always visible) ---
        st.header("Select the Use Case")
        workspace_path = "../workspace"
        if os.path.exists(workspace_path) and os.path.isdir(workspace_path):
            # A use case is any folder containing a usecase.md file
            entries = [d for d in os.listdir(workspace_path) if os.path.isdir(os.path.join(workspace_path, d))]
            saved_use_cases = []
            for d in entries:
                if os.path.isfile(os.path.join(workspace_path, d, "usecase.md")):
                    saved_use_cases.append(d)

            if not saved_use_cases:
                st.info("No saved use cases found in the workspace directory.")
            else:
                def on_uc_change():
                    st.session_state.selected_use_case = st.session_state.use_case_selector
                    all_uc = [d for d in os.listdir("../workspace") if os.path.isfile(os.path.join("../workspace", d, "usecase.md"))]
                    if st.session_state.use_case_selector in all_uc:
                        st.session_state.selected_use_case_index = all_uc.index(st.session_state.use_case_selector)

                uc_index = st.session_state.get('selected_use_case_index', 0)
                st.selectbox(
                    "Select a use case",
                    saved_use_cases,
                    index=uc_index if uc_index < len(saved_use_cases) else 0,
                    key="use_case_selector",
                    on_change=on_uc_change
                )
                if 'use_case_selector' in st.session_state:
                    st.session_state.selected_use_case = st.session_state.use_case_selector
        else:
            st.warning("The 'workspace' directory does not exist.")

    elif selected_uc_tab == "Analysis":
        selected_use_case = st.session_state.get('selected_use_case')

        if st.session_state.get('last_analysis_status'):
            status_info = st.session_state.last_analysis_status
            if status_info['status'] == 'success':
                st.success(status_info['message'])
            elif status_info['status'] == 'error':
                st.error(status_info['message'])
            st.session_state.last_analysis_status = None

        if selected_use_case:
            uc_path = os.path.join("../workspace", selected_use_case)

            st.header("Run Analysis")

            # Load $USE_CASE commands from commands.md
            command_map = {}
            try:
                with open("commands.md", 'r', encoding='utf-8') as f:
                    for line in f:
                        if line.strip():
                            parts = [p.strip() for p in line.strip().split(',', 2)]
                            if len(parts) >= 2:
                                name = parts[0]
                                command_template = parts[1]
                                output_file = parts[2] if len(parts) > 2 else None
                                if "$USE_CASE" in command_template:
                                    command = command_template.replace("$USE_CASE", f"'$(cat usecase.md)'")
                                    # We will run in uc_path so relative outputs go there
                                    command_map[name] = (command, output_file)
            except FileNotFoundError:
                command_map["Error"] = ("echo 'commands.md not found'", None)

            if not command_map:
                st.warning("No Product Use Case commands found in commands.md (missing $USE_CASE).")
            else:
                if not st.session_state.get('command_is_running'):
                    selected_command_name = st.selectbox("Select an analysis to run", list(command_map.keys()))
                    run_button = st.button("Run Analysis")

                    if run_button and selected_command_name:
                        command_to_run, output_file = command_map[selected_command_name]

                        should_run_command = True
                        if output_file:
                            report_file_path = os.path.join(uc_path, output_file)
                            if os.path.exists(report_file_path):
                                st.info(f"Report '{output_file}' already exists for this use case. Analysis not required.")
                                should_run_command = False

                        if should_run_command:
                            logging.info(f"Starting use-case analysis: {selected_command_name} on use case: {selected_use_case}")

                            st.session_state.command_is_running = True
                            st.session_state.command_log = f"$ Go get a coffee while the sentient toasters work their magic\n"
                            st.session_state.command_q = queue.Queue()
                            st.session_state.command_return_code = None
                            st.session_state.selected_command_name_running = selected_command_name
                            st.session_state.repo_path_running = uc_path
                            st.session_state.target_entity_running = st.session_state.get('selected_use_case')

                            process = subprocess.Popen(
                                command_to_run, shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                text=True, cwd=uc_path, bufsize=1, universal_newlines=True, preexec_fn=os.setsid
                            )
                            st.session_state.current_process = process
                            thread = threading.Thread(target=run_command_in_thread, args=(process, st.session_state.command_q))
                            thread.daemon = True
                            thread.start()
                            st.session_state.command_thread = thread

            if st.session_state.get('command_is_running'):
                show_analysis_progress()
        else:
            st.info("Please select a use case first (Use Case tab).")

    elif selected_uc_tab == "Results":
        selected_use_case = st.session_state.get('selected_use_case')
        if selected_use_case:
            uc_path = os.path.join("../workspace", selected_use_case)
            st.header("View Use Case Reports")

            if not os.path.isdir(uc_path):
                st.warning("Selected use case directory not found.")
            else:
                # Show any .md except the source usecase.md
                md_files = [f for f in os.listdir(uc_path) if f.endswith('.md') and f != 'usecase.md']
                if not md_files:
                    st.info("No generated markdown reports found in this use case folder.")
                else:
                    selected_md = st.selectbox("Select a report to view", md_files)
                    if selected_md:
                        md_path = os.path.join(uc_path, selected_md)
                        try:
                            with open(md_path, 'r', encoding='utf-8') as f:
                                md_content = f.read()

                            parts = re.split(r"(```mermaid\n.*?\n```)", md_content, flags=re.DOTALL)
                            for part in parts:
                                if part.strip().startswith("```mermaid"):
                                    mermaid_code = part.strip().replace("```mermaid", "").replace("```", "")
                                    st_mermaid(mermaid_code)
                                else:
                                    st.markdown(part, unsafe_allow_html=True)
                        except Exception as e:
                            st.error(f"Error reading markdown file: {e}")
        else:
            st.info("Please select a use case first (Use Case tab).")



# --- Tabs (Repo Analysis only) ---
if analysis_type == "Repo Analysis":
    selected_tab = ui.tabs(options=tab_options, default_value=default_tab)

    if selected_tab == "Repository":
        with st.expander("Clone a new Repository"):
            clone_method = st.radio("Clone Method", ["From URL", "From Azure DevOps Components"])

            repo_url = ""
            project_name = ""
            organization_name = ""

            if clone_method == "From URL":
                repo_url = st.text_input("Repository URL")
            else:
                organization_name = st.text_input("Organization Name")
                project_name = st.text_input("Project Name")
                repo_url = st.text_input("Repository Name")

            branch_name = st.text_input("Branch (optional)")
            pat_token = st.text_input("Azure DevOps PAT (optional)", type="password")
            proxy_url = st.text_input("Proxy URL (optional)")

            clone_button = st.button("Clone")

            if clone_button and repo_url:
                try:
                    if clone_method == "From Azure DevOps Components":
                        repo_name = repo_url
                        clone_url = f"https://dev.azure.com/{organization_name}/{project_name}/_git/{repo_name}"
                    else:
                        repo_name = repo_url.split("/")[-1].replace(".git", "")
                        clone_url = repo_url

                    clone_path = os.path.join("../workspace", repo_name)
                    
                    if pat_token and "dev.azure.com" in clone_url:
                        clone_url = clone_url.replace("https://", f"https://{pat_token}@")
                        st.info("Using Personal Access Token for Azure DevOps.")

                    if os.path.isdir(clone_path):
                        st.warning(f"Directory {clone_path} already exists. Skipping clone.")
                    else:
                        if proxy_url:
                            os.system(f"git config --global http.proxy {proxy_url}")
                            os.system(f"git config --global https.proxy {proxy_url}")
                            st.info(f"Using proxy: {proxy_url}")
                        with st.spinner(f"Cloning repository from {clone_url}..."):
                            kwargs = {}
                            if branch_name:
                                kwargs['branch'] = branch_name
                            git.Repo.clone_from(clone_url, clone_path, **kwargs)
                        st.success(f"Repository cloned successfully into {clone_path}")
                        
                        # --- Auto-select the cloned repo ---
                        cloned_repos_list = [d for d in os.listdir("../workspace") if os.path.isdir(os.path.join("../workspace", d))]
                        if repo_name in cloned_repos_list:
                            st.session_state.selected_repo_index = cloned_repos_list.index(repo_name)
                        st.rerun()

                except Exception as e:
                    st.error(f"An error occurred during cloning: {e}")

        # --- 2. List Cloned Repositories and Run Commands ---
        st.header("Select the Repository")

        workspace_path = "../workspace"
        if os.path.exists(workspace_path) and os.path.isdir(workspace_path):
            cloned_repos = [d for d in os.listdir(workspace_path) if os.path.isdir(os.path.join(workspace_path, d)) and not os.path.isfile(os.path.join(workspace_path, d, "usecase.md"))]
            
            if not cloned_repos:
                st.info("No cloned repositories found in the workspace directory.")
            else:
                def on_repo_change():
                    st.session_state.selected_repo = st.session_state.repo_selector
                    cloned_repos = [d for d in os.listdir("../workspace") if os.path.isdir(os.path.join("../workspace", d)) and not os.path.isfile(os.path.join("../workspace", d, "usecase.md"))]
                    if st.session_state.repo_selector in cloned_repos:
                        st.session_state.selected_repo_index = cloned_repos.index(st.session_state.repo_selector)

                # Use the index from session_state, then reset it
                repo_index = st.session_state.get('selected_repo_index', 0)
                st.selectbox(
                    "Select a repository",
                    cloned_repos,
                    index=repo_index if repo_index < len(cloned_repos) else 0,
                    key="repo_selector",
                    on_change=on_repo_change
                )
                # The script will rerun on change, and the caption will be updated.
                if 'repo_selector' in st.session_state:
                    st.session_state.selected_repo = st.session_state.repo_selector

        else:
            st.warning("The 'workspace' directory does not exist.")

    elif selected_tab == "Analysis":
        selected_repo = st.session_state.get('selected_repo')

        if st.session_state.get('last_analysis_status'):
            status_info = st.session_state.last_analysis_status
            if status_info['status'] == 'success':
                st.success(status_info['message'])
            elif status_info['status'] == 'error':
                st.error(status_info['message'])
            st.session_state.last_analysis_status = None # Clear after displaying

        if selected_repo:
            repo_path = os.path.join("../workspace", selected_repo)

            # --- New: Command Execution Section ---
            st.header("Run Analysis")
            
            # Load commands from commands.md and replace $REPOSITORY
            command_map = {}
            try:
                with open("commands.md", 'r', encoding='utf-8') as f:
                    relative_repo_path = os.path.join("workspace", selected_repo) + os.sep
                    for line in f:
                        if line.strip():
                            parts = [p.strip() for p in line.strip().split(',', 2)]
                            if len(parts) >= 2:
                                name = parts[0]
                                command_template = parts[1]
                                output_file = parts[2] if len(parts) > 2 else None
                                command = command_template.replace("$REPOSITORY", relative_repo_path)
                                command_map[name] = (command, output_file)

            except FileNotFoundError:
                command_map["Error"] = ("echo 'commands.md not found'", None)
            
            if not command_map:
                st.warning("No valid commands found in commands.md.")
            else:
                # Hide controls when a command is running
                if not st.session_state.get('command_is_running'):
                    selected_command_name = st.selectbox("Select an analysis to run", list(command_map.keys()))
                    run_button = st.button("Run Analysis")

                    # Check if docs server is already running for this repo
                    repo_port = st.session_state.repo_ports.get(repo_path)
                    if repo_port and repo_port in st.session_state.running_servers and st.session_state.running_servers[repo_port].poll() is None:
                        st.success("Documentation server is running.")
                        _, col, _ = st.columns([1, 2, 1])
                        with col:
                            st.link_button("View Documentation", url=f"http://localhost:{repo_port}", use_container_width=True)

                    if run_button and selected_command_name:
                        command_to_run, output_file = command_map[selected_command_name]

                        should_run_command = True
                        if output_file:
                            report_file_path = os.path.join(repo_path, output_file)
                            if os.path.exists(report_file_path):
                                st.info(f"Report '{output_file}' already exists for this repository. Analysis not required.")
                                should_run_command = False
                        
                        if should_run_command:
                            logging.info(f"Starting analysis: {selected_command_name} on repo: {selected_repo}")

                            # --- Start command execution state ---
                            st.session_state.command_is_running = True
                            st.session_state.command_log = f"$ Go get a coffee while the sentient toasters work their magic\n"
                            st.session_state.command_q = queue.Queue()
                            st.session_state.command_return_code = None
                            st.session_state.selected_command_name_running = selected_command_name
                            st.session_state.repo_path_running = repo_path
                            st.session_state.target_entity_running = st.session_state.get('selected_repo')

                            # This logic handles "re-running" the docs server.
                            # It also handles the case where the command *generates* the docs for the first time.
                            if "documentation" in selected_command_name.lower():
                                ra_path = os.path.join(repo_path, '_ra')
                                if os.path.isdir(ra_path):
                                    st.info("Starting/Restarting documentation server...")
                                    # Assign a port to the repo if it doesn't have one
                                    if repo_path not in st.session_state.repo_ports:
                                        st.session_state.repo_ports[repo_path] = st.session_state.next_port
                                        st.session_state.next_port += 1
                                    repo_port = st.session_state.repo_ports[repo_path]
                                    start_docs_server(repo_path, repo_port)
                                    st.session_state.command_is_running = False # It's not a long-running fg task
                                else:
                                    # Run the command to generate the docs first
                                    st.info("Documentation not generated yet. Running generation command...")
                                    process = subprocess.Popen(
                                        command_to_run, shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                        text=True, cwd='../', bufsize=1, universal_newlines=True, preexec_fn=os.setsid
                                    )
                                    st.session_state.current_process = process
                                    thread = threading.Thread(target=run_command_in_thread, args=(process, st.session_state.command_q))
                                    thread.daemon = True
                                    thread.start()
                                    st.session_state.command_thread = thread
                            else:
                                process = subprocess.Popen(
                                    command_to_run, shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                    text=True, cwd='../', bufsize=1, universal_newlines=True, preexec_fn=os.setsid
                                )
                                st.session_state.current_process = process
                                thread = threading.Thread(target=run_command_in_thread, args=(process, st.session_state.command_q))
                                thread.daemon = True
                                thread.start()
                                st.session_state.command_thread = thread

            # This block will now handle rendering the logs for a running command
            if st.session_state.get('command_is_running'):
                show_analysis_progress()

        else:
            st.info("Please select a repository first.")

    elif selected_tab == "Results":
        selected_repo = st.session_state.get('selected_repo')
        if selected_repo:
            repo_path = os.path.join("../workspace", selected_repo)
            
            # --- View Markdown Files ---
            st.header("View Generated Reports")
            
            allowed_md_files = ["ra-overview.md", "ra-obsolescence.md", "ra-security.md", "ra-migrate.md","ra-risk.md"]
            
            report_names = ["Architecture Overview", "Obsolescence Risk Assessment", "Security Assessment","Cross-Language Migration","Risk Graph"]
            
            # Create mapping between report names and filenames
            file_to_report_map = dict(zip(allowed_md_files, report_names))
            report_to_file_map = dict(zip(report_names, allowed_md_files))
            
            md_files = [f for f in os.listdir(repo_path) if f in allowed_md_files]
            
            if not md_files:
                st.info("No designated markdown files found in the root of this repository.")
            else:
                # Get available report names for files that actually exist
                available_report_names = [file_to_report_map[f] for f in md_files]
                
                selected_report_name = st.selectbox("Select a report to view", available_report_names)
                if selected_report_name:
                    # Map the selected report name back to the actual filename
                    selected_md_file = report_to_file_map[selected_report_name]
                    md_path = os.path.join(repo_path, selected_md_file)
                    try:
                        with open(md_path, 'r', encoding='utf-8') as f:
                            md_content = f.read()
                        
                        # Split markdown by mermaid blocks and render accordingly
                        parts = re.split(r"(```mermaid\n.*?\n```)", md_content, flags=re.DOTALL)
                        for part in parts:
                            if part.strip().startswith("```mermaid"):
                                mermaid_code = part.strip().replace("```mermaid", "").replace("```", "")
                                st_mermaid(mermaid_code)
                            else:
                                st.markdown(part, unsafe_allow_html=True)

                    except Exception as e:
                        st.error(f"Error reading markdown file: {e}")
        else:
            st.info("Please select a repository first.")
