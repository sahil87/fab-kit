# Brief: Enhance SRAD Confidence Scoring

**Change**: 260212-f9m3-enhance-srad-fuzzy
**Created**: 2026-02-12
**Status**: Draft

## Origin

User description:
> Enhance SRAD confidence scoring with fuzzy dimension evaluation, validated penalty weights through sensitivity analysis, and dynamic thresholds based on change type. Current binary high/low classification loses granularity that fuzzy logic would preserve. Research shows 45% of MCDA studies use fuzzy set theory for uncertainty handling, and small weight changes can dramatically affect outcomes. Keep SRAD's 4-dimension framework (Signal, Reversibility, Agent Competence, Disambiguation) but add per-dimension scoring on 0-100 scale instead of binary, validate current 0.3/1.0 penalties using historical change data, and test whether 3.0 threshold correlates with actual need for human intervention. References: PMC4544539 on fuzzy MCDA approaches, ScienceDirect S266618882600016X on MCDA reliability, and edge-case Medium article on supervised autonomy frameworks for 2026. Implementation will preserve SRAD's interpretability and efficiency while adding measurable improvements through gradual scoring rather than discrete grades.

**Research Context**: Based on comprehensive research comparing SRAD to state-of-the-art MCDA (Multi-Criteria Decision Analysis) approaches, fuzzy logic, and supervised autonomy frameworks used in 2025-2026 agentic AI systems.

## Why

The current SRAD framework uses binary (high/low) evaluation for each dimension, which loses nuance when decisions fall in the middle range. Research on MCDA uncertainty handling shows that **fuzzy set theory is the most common approach (45% of studies)** for handling ambiguous decision boundaries, and that **small weight changes can dramatically affect outcomes**, suggesting our fixed 0.3/1.0 penalties may be suboptimal.

Additionally, the fixed 3.0 threshold for `/fab-fff` applies uniformly regardless of change risk profile (bugfix vs. new feature vs. refactor), which may be overly conservative for low-risk changes or too permissive for high-risk ones.

This enhancement preserves SRAD's strengths (domain-specific design, interpretability, efficiency) while addressing its key weaknesses through gradual scoring, empirical weight validation, and context-sensitive thresholds.

## What Changes

- **Add per-dimension fuzzy scoring** (0-100 continuous scale) instead of binary high/low classification for S, R, A, D dimensions
- **Validate penalty weights** (currently 0.3 for Confident, 1.0 for Tentative) via sensitivity analysis on historical change data from `fab/changes/**/.status.yaml` (scoring computed by `fab/.kit/scripts/lib/calc-score.sh`)
- **Test threshold calibration** by correlating the 3.0 threshold with actual human intervention needs across completed changes
- **Introduce dynamic thresholds** based on change type categorization (bugfix, feature, refactor, architecture)
- **Preserve linear formula structure** and existing 4-dimension framework (no architectural changes)
- **Document research findings** and methodology in `docs/specs/srad.md` with academic references
- **Write comprehensive SRAD scoring test cases** in `src/lib/calc-score/test.sh` covering fuzzy dimension scoring, dynamic thresholds, weight sensitivity, and edge cases (existing suite covers the current binary formula but not the proposed enhancements)
- **Provide backward compatibility** through optional feature flag or gradual rollout strategy

## Affected Docs

### New Docs
None — this enhances an existing framework rather than introducing a new one.

### Modified Docs
- `docs/specs/skills.md`: Update SRAD dimension evaluation from binary to fuzzy scoring (0-100 scale)
- `docs/specs/srad.md`: Add research findings section, fuzzy scoring methodology, sensitivity analysis results, dynamic threshold tables, and academic references (PMC4544539, ScienceDirect S266618882600016X, supervised autonomy frameworks)
- `fab/.kit/skills/_context.md`: Note the enhanced SRAD scoring in the context-loading section

### Removed Docs
None

## Impact

**Affected Files**:
- `docs/specs/srad.md` — Primary SRAD specification (formula, scoring, worked examples)
- `fab/.kit/scripts/lib/calc-score.sh` — Scoring computation (formula, grade counting, `.status.yaml` writes)
- `fab/.kit/skills/_context.md` — SRAD framework definition (dimension evaluation methodology)
- `src/lib/calc-score/test.sh` — Comprehensive test suite (needs new cases for fuzzy scoring, dynamic thresholds, weight sensitivity)
- `fab/.kit/skills/fab-new.md`, `fab/.kit/skills/fab-continue.md` — Planning skills that apply SRAD
- Historical `.status.yaml` files — Data source for validation (read-only, used for analysis)

**Affected Systems**:
- `calc-score.sh` — the shell script that implements the scoring formula, parses Assumptions tables, and writes the confidence block to `.status.yaml`; fuzzy scoring and dynamic thresholds would require changes here
- `/fab-fff` gate threshold logic (potentially context-sensitive)
- `.status.yaml` schema (may need additional fields for fuzzy dimension scores)

**User Experience**:
- More nuanced confidence scoring that captures middle-ground decisions
- Data-validated penalty weights that better reflect actual ambiguity impact
- Potentially different thresholds for different change types (safer autonomous execution)
- Users can understand dimension scores at finer granularity (0-100 vs. just "high/low")

## Open Questions

None — all decision points resolved via SRAD analysis with Certain or Confident grades. Implementation details for the 0-100 scale (e.g., specific fuzzy membership functions, discretization if needed) are deferred to spec generation phase per user request.

## Assumptions

| # | Grade | Decision | Rationale |
|---|-------|----------|-----------|
| 1 | Confident | Use continuous 0-100 scale for dimension scoring | User specified "0-100 scale" and research supports gradual scoring over binary; specific implementation details to be discussed during spec phase per user request |
| 2 | Confident | Derive change type categories from existing change patterns | Existing changes in `fab/changes/` provide sufficient examples to infer bugfix/feature/refactor/architecture categories |
| 3 | Confident | Implement backward compatibility via gradual rollout | Existing changes have binary scores; gradual migration strategy (feature flag or parallel scoring) prevents breaking existing workflows |

3 assumptions made (3 confident, 0 tentative). Run /fab-clarify to review.

## References

**Academic & Industry Research**:
- [PMC4544539](https://pmc.ncbi.nlm.nih.gov/articles/PMC4544539/): "A Review and Classification of Approaches for Dealing with Uncertainty in Multi-Criteria Decision Analysis for Healthcare Decisions" — Fuzzy set theory most common (45%), probabilistic approaches (15%)
- [ScienceDirect S266618882600016X](https://www.sciencedirect.com/science/article/pii/S266618882600016X): "Multi-Criteria Decision Analysis (MCDA) for sustainability assessment—How reliable are the results?" — Small weight changes dramatically affect outcomes, sensitivity analysis essential
- [Supervised Autonomy Framework 2026](https://edge-case.medium.com/supervised-autonomy-the-ai-framework-everyone-will-be-talking-about-in-2026-fe6c1350ab76): Dynamic thresholds and "volume dial" control over binary gates
- [Measuring Confidence in LLM Responses](https://medium.com/@georgekar91/measuring-confidence-in-llm-responses-e7df525c283f): Confidence-based routing patterns in production LLM systems
- [MCDA Weighting Methods Comparison](https://becarispublishing.com/doi/10.2217/cer-2018-0102): Comparison of weighting schemes in healthcare MCDA

**Methodology**: Multi-Criteria Decision Analysis (MCDA) with fuzzy logic uncertainty handling, sensitivity analysis for weight validation, empirical threshold calibration via historical correlation analysis.
