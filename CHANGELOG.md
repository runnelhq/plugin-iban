# Änderungen

## v0.2.1 (2026-11-26)

- SDK `v0.2.1`: Anfragen mit ungültiger ID (weder ganze Zahl noch Text) beantwortet das Plugin mit `-32600` und
  `"id": null`; leere Listen gehen als `[]` an Runnel (ADR 0019, Nachtrag 2026-11-26).
- Release-Archive nur noch für Windows und Linux (x64), wie Runnel selbst; die Archive für ARM und macOS entfallen.

## v0.2.0 (2026-11-25)

- SDK `v0.2.0`: Die Bausteine stehen in Runnel (ab runnel#236) in der eigenen Gruppe „Plugin: iban“ der Palette.
- Installationsanleitung im README; das README liegt jetzt in jedem Archiv (`plugins/iban/README.md`) und ist im
  Editor lesbar.

## v0.1.0 (2026-11-25)

- Erste Version: `iban.validate` (89 Länder nach dem IBAN-Register von SWIFT), `iban.validate-bic`, `iban.create`.
- Transform `iban.validate` für Pipelines: Fehler-Link mit `_error`/`_errorField` oder Ziel `invalid`, neue Felder für
  Land, Bankleitzahl und Filiale, Schreibweise, `transform.init` für die Prüfung.
- Release-Archive mit erzeugten Lizenzhinweisen und SBOM (CycloneDX) je System; SDK `v0.1.0`.
