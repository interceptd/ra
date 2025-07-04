# Product Requirements Document (PRD)

## LLM Insights Command for repo-analyzer

### **Document Information**

- **Product Name**: repo-analyzer LLM Insights
- **Version**: 0.2
- **Status**: Draft

---

## **1. Executive Summary**

### **1.1 Overview**

Add a new `insights` command to the repo-analyzer tool that leverages the existing analysis outputs to generate comprehensive AI-powered insights about any software project. This command will use a five-stage processing pipeline with intelligent preprocessing to contact OpenAI-compatible LLMs and produce architectural blueprints, detailed documentation, and modernization recommendations.

### **1.2 Business Value**

- **Automated Documentation**: Generate comprehensive project documentation automatically
- **Architecture Visualization**: Create visual diagrams of complex codebases
- **Modernization Guidance**: Provide actionable recommendations for improving codebases
- **Universal Applicability**: Works with any programming language or project type
- **Cost Efficiency**: Reduce manual documentation and code review time by 70-80%

### **1.3 Success Metrics**

- **Time Savings**: Reduce documentation creation time from days to minutes
- **Accuracy**: 90%+ accuracy in architectural representation
- **Adoption**: 80% of analyze command users also use insights command
- **User Satisfaction**: 4.5/5 rating for generated insights quality

---

## **2. Product Overview**

### **2.1 Problem Statement**

Software projects lack comprehensive, up-to-date documentation and architectural insights. Manual documentation is time-consuming, often outdated, and inconsistent across projects. Developers need automated tools that can understand, document, and suggest improvements for any codebase.

### **2.2 Solution**

A new `insights` command that:

1. **Consumes** existing analyzer outputs (gitingest + repomix)
2. **Processes** them through a five-stage intelligent pipeline (Discover → Distill → Architecture → Documentation → Modernization)
3. **Generates** architectural diagrams, documentation, and modernization recommendations
4. **Outputs** results in multiple formats (Markdown, JSON, HTML)

### **2.3 Target Users**

- **Primary**: Software developers and architects
- **Secondary**: Technical writers, project managers, code reviewers
- **Tertiary**: DevOps engineers, consultants, auditors

---

## **3. Functional Requirements**

### **3.1 Core Command Structure**

```bash
repo-analyzer insights [path] [options]
```

### **3.2 Command Options**

```bash
# Basic usage
repo-analyzer insights                    # Current directory
repo-analyzer insights ./my-project      # Specific path
repo-analyzer insights --from-analysis ./analysis-results  # Use existing analysis

# LLM Configuration
repo-analyzer insights --api-key $OPENAI_API_KEY
repo-analyzer insights --model gpt-4o
repo-analyzer insights --base-url https://api.openai.com/v1
repo-analyzer insights --temperature 0.3

# Output Configuration
repo-analyzer insights --output ./insights-output
repo-analyzer insights --format markdown,json,html
repo-analyzer insights --no-diagrams
repo-analyzer insights --skip-modernization

# Pipeline Control
repo-analyzer insights --stages discover,distill,architecture,documentation,modernization
repo-analyzer insights --stage-only architecture
repo-analyzer insights --resume-from distill
repo-analyzer insights --skip-discover  # Use existing distilled payload
repo-analyzer insights --max-chunks 20  # Override default chunk limit

# Advanced Options
repo-analyzer insights --project-name "MyProject"
repo-analyzer insights --project-type "web-app"
repo-analyzer insights --context-limit 32000
repo-analyzer insights --parallel-requests 3
```

### **3.3 Five-Stage Pipeline with Discover & Distill**

#### **Stage 0: Discover (Preprocessing)**

- **Input**: Raw gitingest + repomix analysis files
- **Process**: Static analysis to harvest raw signals (offline, deterministic, cheap)
- **Output**: Language census, dependency graph, entry points, metrics
- **Key Operations**:
  - Language detection from shebangs and file extensions
  - Build/entry file identification (go.mod, package.json, main(), etc.)
  - Static dependency graph via AST parsing
  - Call-graph and key symbol scanning
  - Comment & README embeddings for semantic labeling
  - Code metrics (SLOC, complexity, churn)

#### **Stage 1: Distill (Content Selection)**

- **Input**: Raw signals from Stage 0
- **Process**: Intelligent content selection and summarization
- **Output**: Distilled payload (≤20 chunks, <2k tokens each)
- **Key Operations**:
  - Component clustering using Louvain community detection (≤25 groups)
  - Entry-point linking and data-store detection
  - Top-K chunk selection (entry files, high fan-in/out, complex files)
  - Local summarization with small LLM (Claude Haiku, ~$0.02 total)
  - Blueprint payload composition for optimal LLM consumption

#### **Stage 2: Architectural Blueprint**

- **Input**: Distilled payload from Stage 1
- **Process**: Generate Mermaid diagrams and architectural explanations using Claude 3.5 Sonnet
- **Output**: `architecture.md`, `architecture.json`, mermaid diagrams
- **Feedback Loop**: Auto-request missing information if confidence < 0.7

#### **Stage 3: Functional Documentation**

- **Input**: Stage 2 output + distilled payload
- **Process**: Create comprehensive functional documentation using Claude 3.5 Sonnet
- **Output**: `documentation.md`, `documentation.json`

#### **Stage 4: Modernization Recommendations**

- **Input**: Stage 3 output + distilled payload
- **Process**: Generate improvement suggestions and best practices using Claude 3.5 Sonnet
- **Output**: `modernization.md`, `modernization.json`, `recommendations.json`

### **3.4 Integration Requirements**

- **Dependency**: Must run after or integrate with existing `analyze` command
- **Auto-execution**: Option to run insights automatically after analysis
- **Chain commands**: `repo-analyzer analyze --with-insights`
- **Resume capability**: Resume from any stage if interrupted

---

## **4. Technical Requirements**

### **4.1 Architecture**

#### **4.1.1 Command Structure**

```
cmd/
├── insights.go          # Main insights command
├── insights_pipeline.go # Pipeline orchestration
├── insights_discover.go # Stage 0: Static analysis and signal harvesting
├── insights_distill.go  # Stage 1: Content selection and clustering
├── insights_llm.go      # LLM client abstraction
├── insights_prompts.go  # Prompt templates (updated for distilled payloads)
└── insights_output.go   # Output formatting
```

#### **4.1.2 Data Flow**

```
Analysis Files → Stage 0 (Discover) → Stage 1 (Distill) → Stage 2 (Architecture) → Stage 3 (Documentation) → Stage 4 (Modernization) → Output Generator
                      ↓                     ↓                           ↓
                 Raw Signals         Distilled Payload         Feedback Loop
                 (Offline)           (≤20 chunks, <2k)        (Auto-refine)
```

### **4.2 LLM Integration**

#### **4.2.1 Supported APIs**

- **OpenAI**: gpt-4o, gpt-4o-mini, gpt-3.5-turbo
- **Anthropic**: claude-3-5-sonnet, claude-3-haiku
- **Local**: Ollama, LM Studio
- **Azure OpenAI**: All OpenAI models via Azure
- **Custom**: Any OpenAI-compatible API

#### **4.2.2 API Client Features**

- **Retry Logic**: Exponential backoff for rate limits
- **Token Management**: Automatic chunking for large inputs
- **Error Handling**: Graceful degradation and recovery
- **Caching**: Optional response caching for development
- **Rate Limiting**: Configurable requests per minute

### **4.3 Configuration Schema**

#### **4.3.1 New Configuration Section**

```yaml
# LLM Insights Configuration
insights:
  # LLM Provider Settings
  llm:
    # Primary LLM for main pipeline stages
    provider: "anthropic" # openai, anthropic, azure, ollama, custom
    api_key: "" # From env var or config
    base_url: "https://api.anthropic.com/v1"
    model: "claude-3-5-sonnet-20241022"
    temperature: 0.3
    max_tokens: 4000
    timeout: 60s

    # Small LLM for distillation stage
    distill_provider: "anthropic"
    distill_model: "claude-3-haiku-20240307"
    distill_temperature: 0.1
    distill_max_tokens: 1000

  # Pipeline Settings
  pipeline:
    auto_run_after_analysis: false
    stages:
      ["discover", "distill", "architecture", "documentation", "modernization"]
    resume_on_failure: true
    parallel_requests: 1

    # Discover stage settings
    discover:
      enable_ast_parsing: true
      enable_embedding: true
      embedding_model: "text-embedding-3-small"
      max_file_size: 100000 # Skip files larger than 100KB

    # Distill stage settings
    distill:
      max_clusters: 25
      max_chunks: 20
      chunk_max_tokens: 2000
      chunk_overlap: 500
      confidence_threshold: 0.7
      enable_feedback_loop: true

  # Output Settings
  output:
    formats: ["markdown", "json"]
    include_diagrams: true
    diagram_format: "mermaid"
    output_dir: "insights-output"

  # Context Management
  context:
    max_tokens: 32000
    chunk_overlap: 500
    smart_chunking: true

  # Prompt Customization
  prompts:
    architecture:
      system_prompt: "As a Principal Software Architect, your task is to analyze the provided distilled codebase payload and generate a high-level architectural overview. The codebase is for a project named [PROJECT_NAME]."
      user_prompt: |
        Your output must consist of two parts:
        1. **Architectural Diagram**: Create a `mermaid graph TD` (top-down) diagram that accurately represents the software's architecture. Include user entry points, core logic/engine, data flows, data sources/sinks, and third-party integrations. Use subgraphs to group related components logically.
        2. **Diagram Explanation**: Provide a brief explanation for each major section and describe the overall workflow from user interaction to final output.

        **DISTILLED CODEBASE PAYLOAD:**
        [DISTILLED_PAYLOAD]
      max_response_tokens: 4000
    documentation:
      system_prompt: "As a Senior Technical Writer, your task is to create detailed functional documentation for the provided project. Use the architectural diagram and explanation from our previous step as the foundation."
      user_prompt: |
        Create a well-structured Markdown document with these sections:
        ## 1. Project Overview
        ## 2. Core Concepts & Components  
        ## 3. Key Workflows & Modes of Operation
        ## 4. Data Management & Persistence
        ## 5. Key Design Choices & Integrations

        Focus on explaining *what* the code does and *why* it's designed that way.

        **CONTEXT FROM PREVIOUS STEP:**
        [ARCHITECTURE_OUTPUT]

        **DISTILLED CODEBASE PAYLOAD:**
        [DISTILLED_PAYLOAD]
      max_response_tokens: 4000
    modernization:
      system_prompt: "As a Staff Engineer performing a code review and architectural design session, your task is to analyze the provided codebase and suggest modernizations."
      user_prompt: |
        Structure your response into these categories with Problem-Solution-Benefits for each:
        1. **Architectural & Code Organization**
        2. **Performance & Concurrency** 
        3. **Configuration & Environment Management**
        4. **Testing & Automation**
        5. **Dependency Management & Security**

        **PROJECT CONTEXT:**
        [DOCUMENTATION_OUTPUT]

        **DISTILLED CODEBASE PAYLOAD:**
        [DISTILLED_PAYLOAD]
      max_response_tokens: 4000
```

### **4.4 Input Processing**

#### **4.4.1 Analysis File Detection**

- **Automatic**: Scan for gitingest and repomix outputs
- **Manual**: Specify analysis directory or files
- **Validation**: Ensure required analysis files exist
- **Fallback**: Run analysis if outputs missing

#### **4.4.2 Content Preparation**

- **Merge**: Combine gitingest and repomix outputs intelligently
- **Filter**: Remove sensitive information (API keys, passwords)
- **Chunk**: Split large content for LLM processing
- **Template**: Format content for prompt templates

### **4.5 Output Generation**

#### **4.5.1 File Structure**

```
insights-output/
├── insights_summary.md           # Executive summary
├── discover/
│   ├── language_census.json     # Language distribution
│   ├── dependency_graph.json    # Raw dependency graph
│   ├── entry_points.json        # Detected entry points
│   ├── metrics.json             # Code metrics (SLOC, complexity)
│   └── embeddings.json          # Comment/README embeddings
├── distill/
│   ├── clusters.json            # Component clusters (≤25 groups)
│   ├── distilled_payload.json   # Final payload for LLM (≤20 chunks)
│   ├── chunk_summaries.json     # Local summarization results
│   └── feedback_requests.json   # Missing information requests
├── architecture/
│   ├── architecture.md          # Architectural blueprint
│   ├── architecture.json        # Structured data
│   └── diagrams/
│       ├── overview.mermaid     # Main architecture diagram
│       └── components.mermaid   # Component diagrams
├── documentation/
│   ├── documentation.md         # Functional documentation
│   ├── documentation.json       # Structured data
│   └── sections/
│       ├── overview.md
│       ├── components.md
│       └── workflows.md
├── modernization/
│   ├── modernization.md         # Improvement recommendations
│   ├── modernization.json       # Structured data
│   └── reports/
│       ├── architecture.md
│       ├── performance.md
│       ├── security.md
│       └── testing.md
└── raw_responses/               # LLM raw outputs for debugging
    ├── stage2_response.json     # Architecture stage
    ├── stage3_response.json     # Documentation stage
    └── stage4_response.json     # Modernization stage
```

#### **4.5.2 Output Formats**

- **Markdown**: Human-readable documentation
- **JSON**: Structured data for programmatic use
- **HTML**: Web-friendly presentation
- **PDF**: Printable reports (future enhancement)

### **4.6 Discover & Distill Approach**

The key innovation in this approach is the **Discover & Distill** preprocessing pipeline that solves the token limit problem for large codebases:

#### **4.6.1 The Problem**

- Raw repository analysis can produce 1M+ tokens of content
- Claude 3.5 Sonnet has ~200K token limit
- Sending full analysis would cost $200-1000+ per repository
- Most content is redundant or low-value for architectural insights

#### **4.6.2 The Solution**

```
Raw Analysis (1M+ tokens) → Discover → Distill → LLM Pipeline (≤40K tokens)
```

#### **4.6.3 Discover Stage (Offline)**

- **Language Census**: Identify primary languages and frameworks
- **Dependency Graph**: Build static call graph via AST parsing
- **Entry Points**: Detect main(), HTTP handlers, CLI commands
- **Metrics**: Calculate complexity, fan-in/out, churn
- **Embeddings**: Vector index of comments/docs for semantic search

#### **4.6.4 Distill Stage (Smart Selection)**

- **Clustering**: Group related files into ≤25 components
- **Ranking**: Select top files per cluster (entry points, high complexity)
- **Summarization**: 50-word abstracts via Claude Haiku (~$0.02)
- **Payload**: JSON structure with clusters, edges, snippets (≤20 chunks)

#### **4.6.5 Benefits**

- **95% Token Reduction**: From 1M+ to ≤40K tokens
- **99% Cost Reduction**: From $200+ to $0.50-2.00 per analysis
- **Quality Preservation**: Maintains architectural signal
- **Scalability**: Works with repositories of any size

---

## **5. Non-Functional Requirements**

### **5.1 Performance**

- **Processing Time**: Complete 5-stage pipeline in under 10 minutes for typical projects
  - Stage 0 (Discover): < 2 minutes (offline processing)
  - Stage 1 (Distill): < 1 minute (local LLM summarization)
  - Stages 2-4 (LLM): < 7 minutes (Claude 3.5 Sonnet processing)
- **Memory Usage**: < 1GB peak memory usage during discover/distill stages
- **Token Efficiency**: 95% reduction in LLM token usage vs. raw analysis
- **Cost Optimization**: ~$0.50-2.00 per large repository vs. $20-100+ without distillation
- **Concurrent Requests**: Support 1-5 parallel LLM requests
- **Caching**: 90% cache hit rate for repeated analysis

### **5.2 Reliability**

- **Error Recovery**: Resume from any stage after failure
- **Retry Logic**: 3 retries with exponential backoff
- **Graceful Degradation**: Continue with partial results if one stage fails
- **Validation**: Validate LLM responses before processing

### **5.3 Security**

- **API Key Protection**: Never log or expose API keys
- **Content Filtering**: Remove sensitive information before LLM processing
- **Local Processing**: Option to use local LLMs for sensitive codebases
- **Rate Limiting**: Prevent API abuse

### **5.4 Usability**

- **Progress Indicators**: Real-time progress for long-running operations
- **Error Messages**: Clear, actionable error messages
- **Help System**: Comprehensive help and examples
- **Configuration**: Sensible defaults with easy customization

---

## **6. Implementation Plan**

### **6.1 Phase 1: Core Infrastructure (Week 1-2)**

- [ ] Create basic command structure
- [ ] Implement LLM client abstraction (dual model support)
- [ ] Add configuration schema with discover/distill settings
- [ ] Create output directory structure

### **6.2 Phase 2: Discover & Distill Implementation (Week 3-4)**

- [ ] Implement Stage 0 (Discover): Static analysis engine
- [ ] Add dependency graph generation and AST parsing
- [ ] Implement Stage 1 (Distill): Component clustering and content selection
- [ ] Add local LLM summarization for chunks
- [ ] Create distilled payload format

### **6.3 Phase 3: LLM Pipeline Implementation (Week 5-6)**

- [ ] Implement Stages 2-4 with updated prompts
- [ ] Add feedback loop for missing information
- [ ] Create prompt templates for distilled payloads
- [ ] Add basic output generation

### **6.4 Phase 4: Advanced Features (Week 7-8)**

- [ ] Add resume capability for all stages
- [ ] Implement parallel processing where applicable
- [ ] Add multiple output formats
- [ ] Create error handling and recovery

### **6.5 Phase 5: Testing and Optimization (Week 9-10)**

- [ ] Comprehensive testing with large repositories
- [ ] Performance optimization for discover/distill stages
- [ ] Cost analysis and token usage optimization
- [ ] User documentation and example workflows

---

## **7. Configuration Examples**

### **7.1 Basic Configuration**

```yaml
insights:
  llm:
    provider: "anthropic"
    model: "claude-3-5-sonnet-20241022"
    temperature: 0.3
    distill_model: "claude-3-haiku-20240307"
  pipeline:
    stages:
      ["discover", "distill", "architecture", "documentation", "modernization"]
  output:
    formats: ["markdown"]
    include_diagrams: true
```

### **7.2 Advanced Configuration**

```yaml
insights:
  llm:
    provider: "anthropic"
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-3-5-sonnet-20241022"
    temperature: 0.3
    max_tokens: 4000
    timeout: 60s
    distill_provider: "anthropic"
    distill_model: "claude-3-haiku-20240307"
  pipeline:
    stages:
      ["discover", "distill", "architecture", "documentation", "modernization"]
    parallel_requests: 3
    resume_on_failure: true
    discover:
      enable_ast_parsing: true
      enable_embedding: true
      max_file_size: 100000
    distill:
      max_clusters: 25
      max_chunks: 20
      confidence_threshold: 0.7
      enable_feedback_loop: true
  output:
    formats: ["markdown", "json", "html"]
    include_diagrams: true
    diagram_format: "mermaid"
    output_dir: "insights-output"
  context:
    max_tokens: 200000 # Claude 3.5 Sonnet context limit
    chunk_overlap: 500
    smart_chunking: true
```

### **7.3 Local LLM Configuration**

```yaml
insights:
  llm:
    provider: "ollama"
    base_url: "http://localhost:11434"
    model: "codestral"
    temperature: 0.3
    # Use local model for distillation too
    distill_provider: "ollama"
    distill_model: "llama3.1:8b"
    distill_temperature: 0.1
  pipeline:
    parallel_requests: 1
    discover:
      enable_ast_parsing: true
      enable_embedding: false # Disable embeddings for local-only setup
    distill:
      max_clusters: 20
      max_chunks: 15
      confidence_threshold: 0.6
  output:
    formats: ["markdown"]
```

---

## **8. Usage Examples**

### **8.1 Basic Usage**

```bash
# Analyze and generate insights
repo-analyzer analyze ./my-project
repo-analyzer insights ./my-project

# Or combined
repo-analyzer analyze ./my-project --with-insights
```

### **8.2 Custom Configuration**

```bash
# Use different model
repo-analyzer insights --model gpt-4o-mini --temperature 0.1

# Skip modernization stage
repo-analyzer insights --stages discover,distill,architecture,documentation

# Use existing analysis (runs discover & distill first)
repo-analyzer insights --from-analysis ./analysis-results

# Use existing distilled payload (skip discover & distill)
repo-analyzer insights --from-distill ./insights-output/distill
```

### **8.3 Advanced Workflows**

```bash
# Resume failed pipeline
repo-analyzer insights --resume-from distill

# Generate only architecture (still needs discover & distill)
repo-analyzer insights --stage-only architecture

# Run only discover and distill stages
repo-analyzer insights --stages discover,distill

# Multiple output formats
repo-analyzer insights --format markdown,json,html

# Skip discover stage and use existing dependency graph
repo-analyzer insights --skip-discover --from-graph ./dependency-graph.json
```

---

## **9. Risks and Mitigation**

### **9.1 Technical Risks**

- **Risk**: LLM API rate limits
  **Mitigation**: Implement exponential backoff and parallel request limiting

- **Risk**: Large codebase context overflow
  **Mitigation**: Intelligent chunking and summarization

- **Risk**: Inconsistent LLM responses
  **Mitigation**: Response validation and retry logic

### **9.2 Business Risks**

- **Risk**: High API costs for large projects
  **Mitigation**: Cost estimation and budget controls

- **Risk**: Sensitive code exposure
  **Mitigation**: Local LLM support and content filtering

- **Risk**: Poor quality outputs
  **Mitigation**: Extensive testing and prompt engineering

---

## **10. Future Enhancements**

### **10.1 Version 2.0 Features**

- **Interactive Mode**: Chat-based interface for refining insights
- **Custom Prompts**: User-defined prompt templates
- **Team Collaboration**: Shared insights and comments
- **Integration**: IDE plugins and CI/CD integration

### **10.2 Advanced Analytics**

- **Trend Analysis**: Track code evolution over time
- **Comparison Mode**: Compare different codebases
- **Metrics Dashboard**: Visual metrics and KPIs
- **Quality Scoring**: Automated code quality assessment

---

## **11. Acceptance Criteria**

### **11.1 Functional Criteria**

- [ ] Successfully processes gitingest + repomix outputs
- [ ] Generates valid Mermaid diagrams
- [ ] Creates comprehensive documentation
- [ ] Provides actionable modernization recommendations
- [ ] Supports multiple LLM providers
- [ ] Handles errors gracefully

### **11.2 Quality Criteria**

- [ ] 90% accuracy in architectural representation
- [ ] Documentation completeness score > 85%
- [ ] Modernization recommendations relevance > 80%
- [ ] User satisfaction rating > 4.5/5

### **11.3 Performance Criteria**

- [ ] Complete 5-stage pipeline in < 10 minutes for typical projects
- [ ] Memory usage < 1GB during discover/distill phases
- [ ] 95% token reduction vs. raw analysis approach
- [ ] 99% uptime for LLM integrations
- [ ] Cost < $2.00 per large repository analysis

---

## **12. Dependencies**

### **12.1 External Dependencies**

- **LLM APIs**: OpenAI, Anthropic, or compatible services
- **Go Libraries**: HTTP client, JSON parsing, template engine
- **Mermaid**: For diagram generation and validation

### **12.2 Internal Dependencies**

- **Existing Commands**: analyze, gitingest, repomix
- **Configuration System**: Viper-based configuration
- **Output System**: Existing output directory management

---

## **13. Success Metrics**

### **13.1 Quantitative Metrics**

- **Adoption Rate**: 80% of analyze command users also use insights
- **Time Savings**: 70-80% reduction in documentation time
- **Accuracy**: 90%+ architectural accuracy
- **Performance**: < 10 minutes for typical projects (5-stage pipeline)

### **13.2 Qualitative Metrics**

- **User Satisfaction**: 4.5/5 rating
- **Documentation Quality**: Comprehensive and actionable
- **Recommendation Relevance**: 80%+ of recommendations are applicable
- **Ease of Use**: Intuitive command structure and configuration

---

## **14. Key Innovation Summary**

This PRD introduces a revolutionary **Discover & Distill** approach that solves the fundamental token limit problem for large repository analysis:

### **14.1 The Challenge**

- Repository analysis can generate 1M+ tokens of content
- LLM context limits prevent processing large codebases
- Direct analysis would cost $200-1000+ per repository
- Most analysis content is redundant for architectural insights

### **14.2 The Solution**

- **Stage 0 (Discover)**: Offline static analysis harvests raw signals
- **Stage 1 (Distill)**: Intelligent clustering and summarization reduces content by 95%
- **Stages 2-4 (LLM)**: Claude 3.5 Sonnet processes distilled payload efficiently
- **Result**: Universal, scalable, cost-effective repository insights

### **14.3 The Impact**

- ✅ **Scalability**: Works with repositories of any size
- ✅ **Cost Efficiency**: 99% cost reduction ($0.50-2.00 vs $200+)
- ✅ **Quality**: Preserves architectural signal while eliminating noise
- ✅ **Speed**: 10-minute end-to-end processing for large codebases
- ✅ **Universality**: Language and framework agnostic

---

This PRD provides a comprehensive roadmap for implementing the LLM insights command that will transform repository analysis into actionable intelligence for any software project.
