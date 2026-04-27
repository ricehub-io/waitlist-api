-- unaimeds: Renaming to _applicants because we're storing a single application
-- per person so its more accurate this way :nerd:

ALTER TABLE founder_applications RENAME TO founder_applicants;