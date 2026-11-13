# Fork workflow

* Keep all development changes for this fork on `combined`.
* Do not create task-specific branches unless the user explicitly requests one.
* Use `combined` as the source revision for development image and chart builds.
* Keep `main` as the existing baseline; it is not the development branch.

This is the user's explicit workflow preference and overrides the parent
instruction to create a new branch for each task. Other parent instructions,
including signed commits and signature verification before pushing, still apply.
