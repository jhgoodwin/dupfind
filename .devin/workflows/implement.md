---
name: implement
description: Methodical implementation of a feature
auto_execution_mode: 2
---

Follow the Engineering Operational Manifesto.

## Risk and Implementation

*   **Prioritize Risk:** Address high-uncertainty components first. Use inexpensive, high-contrast experiments to resolve "unknown unknowns" before committing to production implementation.
*   **Restrict Scope:** Limit implementation to the smallest viable delta. Minimize the change-set to reduce the verification surface area and prevent cumulative validation debt.
*   **Transition Types:** Use loose typing for requirement discovery. Once the data structure is understood, codify it into strict, structural types.

## Testing and Validation

*   **Enforce Contract Hierarchy:**
    *   **Unit Level:** Test pure units to prove the contract. Prohibit the re-validation of integration logic at this level.
    *   **Integration/E2E Level:** Execute realistic user journeys to isolate glue errors and contract gaps.
*   **Reduce Testing Dimensions:** Minimize the test surface area. Implement cheap, project-level assertions to catch systemic failures.
*   **Optimize Test Data:** Use the smallest possible dataset required to trigger a logic path. Prohibit the use of default constants if smaller values prove the logic with lower latency.

## Code Evolution and Maintenance

*   **Apply Parallel Change:** Prohibit in-place refactoring of working code. Implement net-new logic and deprecate references to the legacy code to ensure atomic cut-overs.
*   **Construct for Correctness:** Replace runtime validation with structural constraints. Use domain primitives to make invalid states structurally impossible.
*   **Optimize for Maintainability:** Balance resource efficiency with legibility. Code must function as documentation, enabling a maintainer with 50% of the author's skill to manage the system without consultation.

## Governance

*   **Execute Principled Pragmatism:** Follow established heuristics. Deviations require a documented rationalization. Revert to the heuristic immediately once the reason for the deviation is resolved.
*   **Definition of Done:** A task is complete only when:
    1.  Logic is structural.
    2.  Contracts are proven.
    3.  Cognitive load for a new maintainer is minimized.
*   **Evidence-Based Pivot:** Prioritize empirical evidence over established perspective. When evidence indicates an error in current logic or direction, pivot immediately. If evidence contradicts a long-established pattern, execute a targeted, simple experiment to resolve the discrepancy before changing the perspective.

## Implementation Steps

1.  **Understand the Problem:** Break down the feature into discrete, testable components.
2.  **Design the Solution:** Create a high-level architecture that adheres to the manifest.
3.  **Implement the Solution:** Write code that follows the manifest's principles.
4.  **Test the Solution:** Verify that the implementation meets the requirements and follows the manifest.
5.  **Document the Solution:** Update the documentation to reflect the new implementation.
6.  **Review the Solution:** Ensure that the implementation is correct and follows the manifest.
7.  **Deploy the Solution:** Deploy the implementation to production.
8.  **Monitor the Solution:** Monitor the implementation to ensure that it is working correctly.
9.  **Iterate on the Solution:** Make improvements to the implementation based on feedback and monitoring.
10. **Quality Assurance:** Do not review your own diff inline. Spawn a fresh `devin` CLI process with `devin --model glm-5-2 -- Use the grey-review skill on <reference documentation>` so the review has no memory of this turn's context. It reviews `git diff`/`git diff --stat` and records approval via `.devin/skills/grey-review/grey-approve.sh` if there are no blocking issues.
11. **Incremental Commit:** If the feature is phased, run `.devin/workflows/phase-commit.sh "<message>"` to commit each phase. It refuses to commit unless `grey-review` has approved the exact tree being committed.
12. **Repeat:** Continue the cycle of understanding, designing, implementing, testing, documenting, reviewing, deploying, monitoring, and iterating until the feature is complete.