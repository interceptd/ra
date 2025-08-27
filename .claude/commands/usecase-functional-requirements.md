As a Senior Product Analyst, your task is to read the product use case description in $USE_CASE and produce a precise, testable **Functional Requirements** specification.

Deliverables (in this order):

1) **Context (≤100 words)**
   - One short paragraph summarizing the use case so a new engineer understands the scope.

2) **User Stories** (bullet list)
   - Format: `As a <role>, I want <capability>, so that <benefit>.`

3) **Functional Requirements** (numbered FR-1, FR-2, ...)
   - Each requirement must be **atomic, observable, and testable**.
   - Include inputs, triggers, main flow, and outputs where relevant.
   - Reference related user stories (e.g., US-3).

4) **Acceptance Criteria** per requirement
   - For each FR-n provide 2–5 Given/When/Then scenarios.

5) **Error & Edge Cases**
   - Enumerate failures, timeouts, rate limits, partial successes, retries, idempotency, etc.

6) **Non-Functional Requirements (NFRs)**
   - Performance (latency/throughput), availability/SLOs, security, privacy, audit, compliance, observability, accessibility, i18n/l10n, scalability.

7) **Assumptions & Open Questions**
   - Explicit assumptions and a checklist of questions for stakeholders.

8) **Traceability Table**
   - Columns: `Req ID | User Story | Acceptance Criteria IDs | Notes`

Formatting rules:
- Use clear headings and numbered lists. Keep language concise and imperative.
- Do **not** include any styles or mermaid unless strictly necessary.

Guardrails:
- Do not generate any HTML, JavaScript, or other asset files.
- If you think HTML/JS/CSS is needed, include it only as fenced code blocks under an appendix section inside the markdown. Do not write separate files.
- Create **exactly one** file named `functional-requirements.md`.
- Overwrite the output file if it already exists.

Save the mermaidjs as markdown on $ARGUMENTSfunctional-requirement.md
