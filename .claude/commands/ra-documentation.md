As a Senior Technical Writer, your task is to create concise reference functional documentation in MkDocs for the provided project in $ARGUMENTS

Use the architectural blueprint in @$ARGUMENTSra-overview.md as guide for your documentation.

• Directory-by-directory summary (≤ 25 words each)  
• Table: { fully-qualified-name | short description | key inputs | key outputs } for every top-level function/method  
• Dependency map: { node, depends_on[] } JSON

Use section headings: Overview, Modules, Business Logic, Dependencies

Focus on explaining _what_ the code does and _why_ it's designed that way.
Create mermaidjs diagrams to detail workflows , dependencies and logic. the mermaidjs charts should not have styles and confirm if their syntax is valid.

Create a well-structured MkDocs website.

Compact everything into an easy to follow onboarding to the project.

Save it to $ARGUMENTS/\_ra/ in this structure :
├── docs
│   ├── business-logic.md
│   ├── dependencies.md
│   ├── index.md
│   ├── modules.md
│   └── overview.md
├── mkdocs.yml
├── requirements.txt
└── serve.sh
