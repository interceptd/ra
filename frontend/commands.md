1. Architecture Overview,claude -p "/ra-overview $REPOSITORY" --dangerously-skip-permissions,ra-overview.md
2. Base Documentation,claude -p "/ra-documentation $REPOSITORY" --dangerously-skip-permissions,\_ra/mkdocs.yml
3. Obsolescence Report,claude -p "/ra-obsolescence $REPOSITORY" --dangerously-skip-permissions,ra-obsolescence.md
   3.1 Risk Graph,claude -p "/ra-risk $REPOSITORY" --dangerously-skip-permissions,ra-risk.md
4. Security Assessment,claude -p "/ra-pentest $REPOSITORY" --dangerously-skip-permissions,ra-security.md
5. Cross-Language Migration,claude -p "/ra-migrate $REPOSITORY" --dangerously-skip-permissions,ra-migrate.md
6. Product Functional Requirements,claude -p "/usecase-functional-requirements $USE_CASE" --dangerously-skip-permissions,functional-requirements.md