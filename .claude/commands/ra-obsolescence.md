As a Principal Software Security Architect, your task is to scan the entire repo $ARGUMENTS for obsolescence risk:

• Deprecated libraries / APIs  
• Secrets or API keys in code
• Anti-patterns (list follows: Singleton-everywhere, naked SQL, threading w/out locks …)  
• Dead code (never referenced)

Limit to top 10 findings, create a markdown table with : issue_type, location, description, severity

save the full report on the file $ARGUMENTSra-obsolescence.md
