# docs/site/

Site-only prose along the **audience axis** — content meant for the public docs site
(shll.ai/fab-kit) that should *not* live in the `README.md` itself (e.g. extended narrative or
examples kept out of the README to keep the pulled slice tight).

This directory implements the user-facing branch of the audience axis from shll.ai's
README-extraction contract (§9): the README slice is the **default** site source; `docs/site/`
is the reserved extension for additions that belong on the site but not in the README.

## ⚠️ Status: §9 pull path is RESERVED / NOT YET IMPLEMENTED

shll.ai's puller (`extract-readme.ts` / `scheduled-readme-refresh.yml` / `ReadmeSlice.astro`)
fetches and renders **only** `README.md` today. **Nothing reads `docs/site/` yet.** Content
placed here will *not* appear on the site until a future shll.ai change ships the §9
pull + render path.

**Consequence — do not strand content here.** Because this directory is not pulled, any
README content moved *into* `docs/site/` would vanish from the site (which can't pull it) while
also leaving the GitHub README. So this directory currently holds **structure + this explainer
only**; no load-bearing README prose has been migrated. Anything placed here later must remain
reachable on GitHub via an absolute blob link (`https://github.com/sahil87/fab-kit/blob/main/docs/site/<file>.md`)
so it is not lost on either surface.

When shll.ai implements §9, real site-only prose can migrate here.
