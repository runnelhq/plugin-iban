# Änderungen

## Unveröffentlicht (v0.1.0)

- Erste Version: `iban.validate` (89 Länder nach dem IBAN-Register von SWIFT), `iban.validate-bic`, `iban.create`.
- Transform `iban.validate` für Pipelines: Fehler-Link mit `_error`/`_errorField` oder Ziel `invalid`, neue Felder für
  Land, Bankleitzahl und Filiale, Schreibweise, `transform.init` für die Prüfung.
