# Änderungen

## v0.2.0 (unveröffentlicht)

- SDK `v0.2.0`: Die Bausteine stehen in Runnel (ab runnel#236) in der eigenen Gruppe „Plugin: iban“ der Palette.
- Installationsanleitung im README; das README liegt jetzt in jedem Archiv (`plugins/iban/README.md`) und ist im
  Editor lesbar.

## v0.1.0 (2026-11-25)

- Erste Version: `iban.validate` (89 Länder nach dem IBAN-Register von SWIFT), `iban.validate-bic`, `iban.create`.
- Transform `iban.validate` für Pipelines: Fehler-Link mit `_error`/`_errorField` oder Ziel `invalid`, neue Felder für
  Land, Bankleitzahl und Filiale, Schreibweise, `transform.init` für die Prüfung.
- Release-Archive mit erzeugten Lizenzhinweisen und SBOM (CycloneDX) je System; SDK `v0.1.0`.
