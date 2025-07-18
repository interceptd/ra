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
    with st.status(f"Running analysis: `{st.session_state.selected_command_name_running}` on `{st.session_state.selected_repo}`...", expanded=True) as status:
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
                
            else:
                time.sleep(1) # The fragment will re-run itself, not the whole app

st.title("DORA")
st.caption("Documentation, Obsolescence, Risk and Architecture Assistant")

if st.session_state.get('selected_repo'):
    st.caption(f"Selected Repository: `{st.session_state.get('selected_repo')}`")

# --- Tabs ---
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
        cloned_repos = [d for d in os.listdir(workspace_path) if os.path.isdir(os.path.join(workspace_path, d))]
        
        if not cloned_repos:
            st.info("No cloned repositories found in the workspace directory.")
        else:
            def on_repo_change():
                st.session_state.selected_repo = st.session_state.repo_selector
                cloned_repos = [d for d in os.listdir("../workspace") if os.path.isdir(os.path.join("../workspace", d))]
                if st.session_state.repo_selector in cloned_repos:
                    st.session_state.selected_repo_index = cloned_repos.index(st.session_state.repo_selector)

            # Use the index from session_state, then reset it
            repo_index = st.session_state.get('selected_repo_index', 0)
            st.selectbox(
                "Select a repository",
                cloned_repos,
                index=repo_index,
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
        
        # # --- Display Risk SVG if it exists ---
        # risk_svg_path = os.path.join(repo_path, 'ra-risk.svg')
        # if os.path.exists(risk_svg_path):
        #     with open(risk_svg_path, "r") as f:
        #         svg_content = f.read()
        #     st.markdown(f"## Risk Graph")
        #     st.markdown(f'<div style="text-align: center;">{svg_content}</div>', unsafe_allow_html=True)

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