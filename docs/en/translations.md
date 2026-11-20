<!-- docs-pair: translations; revision: 1 -->

[English](translations.md) | [Español](../es/translations.md) | [Index](README.md)

# Translations

English is the editorial reference. English and Spanish are maintained together.
Other languages are welcome through reviewed contributions; they are not listed
as available before their files exist.

## Current language status

| Language | Directory | Status |
| --- | --- | --- |
| English (`en`) | [English index](README.md) | Reference, revision 1. |
| Español (`es`) | [Índice en español](../es/README.md) | Complete counterpart, revision 1. |

Matching revision markers record the intended baseline; they are not proof of
translation quality. Review meaning, technical accuracy and tone manually.

## Updating English or Spanish

Update both versions when shared meaning changes and increase their common
`revision` in the `docs-pair` comment. Keep the identifier unchanged. Corrections
that only improve spelling need not change the revision. The documentation checker
detects missing paired files, missing navigation and inconsistent revisions.

Translate explanations and accessible descriptions. Preserve paths, protocol
fields, code identifiers and executable commands. Use the [glossary](glossary.md)
and describe unfamiliar terms instead of introducing conflicting translations.
The official license text remains unchanged; translated licensing explanations
are informative only.

## Adding a language

1. Open a translation proposal with a BCP 47 language tag, scope and available reviewers.
2. Add documents under the matching language directory, using the English filenames.
3. Include a local index and links to the source. Record the source revision in
   each document; mark partial or unreviewed work visibly at its beginning.
4. Start with the project introduction, contribution route and safety limitations.
   Do not translate only promotional text while omitting the design-stage notice.
5. Update the language-status table in both maintained guides. Label partial
   coverage clearly; link only to content that exists.
6. Request a reviewer who understands that language and the relevant technical topic.

Machine translation can be a draft, but must be disclosed as unreviewed until a
language reviewer checks it. Contributors may help with a single document; they
do not need to promise permanent maintenance of an entire language.
