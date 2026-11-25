# plugin-iban – IBAN und BIC für Runnel

Offizielles [Runnel](https://github.com/runnelhq/runnel)-Plugin: prüft IBANs und BICs und bildet IBANs aus Bankleitzahl
und Kontonummer – als Actions in Workflows und als Transform in Pipelines. Gebaut mit dem [Go-SDK](https://github.com/runnelhq/sdk-go). Es rechnet nur lokal – nichts wird
gesendet, es braucht keine Connection.

| | |
|---|---|
| Namensraum | `iban` |
| Actions | `iban.validate`, `iban.validate-bic`, `iban.create` |
| Transforms | `iban.validate` (mit Fehler-Link, `copies` möglich) |
| Runnel | Actions ab S49c, Transform ab S50b |
| Lizenz | MIT |

## Installieren

Das Plugin gehört in den Ordner `plugins/` eines Runnel-Projekts. Runnel startet es erst, wenn es auf dem Rechner für
dieses Projekt freigegeben ist. Beispiel unter Windows mit dem Projekt `D:\flows`:

**1. Archiv laden und prüfen.** Von den [Releases](https://github.com/runnelhq/plugin-iban/releases) das Archiv für das
System laden, dazu `SHA256SUMS`:

| System | Archiv |
|---|---|
| Windows (Intel/AMD) | `runnel-plugin-iban_<version>_windows_amd64.zip` |
| Windows (ARM) | `runnel-plugin-iban_<version>_windows_arm64.zip` |
| Linux | `runnel-plugin-iban_<version>_linux_amd64.zip` bzw. `…_linux_arm64.zip` |
| macOS (Apple-Chip) | `runnel-plugin-iban_<version>_darwin_arm64.zip` |

Das Archiv eines anderen Systems hilft nicht: Die `plugin.yaml` darin nennt nur das Programm ihres Systems.

```powershell
Get-FileHash .\runnel-plugin-iban_0.2.0_windows_amd64.zip -Algorithm SHA256
# muss mit der Zeile in SHA256SUMS übereinstimmen
```

Wer mehr Sicherheit will, prüft zusätzlich die Herkunft mit der GitHub-CLI:
`gh attestation verify runnel-plugin-iban_0.2.0_windows_amd64.zip --repo runnelhq/plugin-iban`. Zu jedem Archiv gibt
es eine SBOM (`.cdx.json`, CycloneDX) mit allen eingebauten Komponenten und ihren Lizenzen.

**2. In das Projekt entpacken,** sodass `plugins/iban/plugin.yaml` entsteht:

```powershell
Expand-Archive .\runnel-plugin-iban_0.2.0_windows_amd64.zip -DestinationPath D:\flows\plugins
```

```text
D:\flows\
├─ project.yaml
└─ plugins\
   └─ iban\
      ├─ plugin.yaml
      ├─ README.md            # diese Anleitung, im Editor lesbar
      ├─ bin\iban.exe
      ├─ LICENSE
      └─ THIRD_PARTY_NOTICES
```

**3. Freigeben.** Runnel zeigt Name, Version, Namensraum, Programm und den Hash des Ordners und fragt nach; `y` gibt frei
(`--yes` für Skripte):

```powershell
runnel plugins trust D:\flows --plugin iban
```

Läuft Runnel als Windows-Dienst, zusätzlich mit `--service` in einer Konsole „Als Administrator“: Die Freigaben des
Dienstes sind getrennt von denen des Benutzers. Im Editor geht die Freigabe auch über den Hinweis „Trust“ am Plugin.

**4. Prüfen:**

```powershell
runnel plugins D:\flows
```

Das Plugin steht dort als `trusted` mit seinen Typen. Danach stehen sie in Workflows und Pipelines und in der Palette des
Editors bereit (Gruppe „Plugin: iban“); `runnel validate D:\flows` prüft die Dateien, die sie nutzen.

**Update:** den Ordner `plugins/iban/` durch den Inhalt des neuen Archivs ersetzen und erneut
`runnel plugins trust` ausführen. Jede Änderung im Ordner hebt die Freigabe auf; bis dahin meldet Runnel das Plugin als
`changed since trusted` und startet keine Läufe, die es brauchen.

**Entfernen:** `runnel plugins untrust D:\flows --plugin iban`, dann den Ordner löschen.

## Actions

### `iban.validate` – IBAN prüfen

Prüft Land, Länge und Aufbau nach dem IBAN-Register von SWIFT (89 Länder) und die Prüfziffern (Modulo 97).
Leerzeichen, Bindestriche, kleine Buchstaben und ein führendes `IBAN` sind erlaubt.

```yaml
- id: check
  type: iban.validate
  with:
    iban: ${IBAN}
    failOnInvalid: true      # Standard false: Ergebnis steht in valid und error
```

Ergebnis (`results.check.output`):

```json
{"valid": true, "iban": "DE89370400440532013000", "formatted": "DE89 3704 0044 0532 0130 00",
 "country": "DE", "checkDigits": "89", "bban": "370400440532013000", "bankCode": "37040044", "branchCode": "",
 "error": ""}
```

Bei einer ungültigen IBAN ist `valid` `false` und `error` nennt den Grund, z. B. `an IBAN of DE has 22 characters,
not 21` oder `the check digits are wrong (mistyped or swapped characters)`. Mit `failOnInvalid: true` scheitert die
Action mit dieser Meldung (Fehlerpfad im Workflow).

Nicht geprüft werden nationale Prüfziffern der Kontonummer und ob Bank oder Konto existieren.

### Transform `iban.validate` – IBANs in Datensätzen prüfen

In Pipelines prüft `iban.validate` ein Feld jedes Datensatzes. Gültige IBANs gehen weiter – auf Wunsch in einheitlicher
Schreibweise und mit Land, Bankleitzahl und Filiale als neuen Feldern; ungültige gehen auf den **Fehler-Link** (mit
`_error` und `_errorField`) oder unverändert an ein Ziel.

```yaml
transforms:
  - id: read
    type: csv.input
    with: { path: accounts.csv }
  - id: check
    type: iban.validate
    with:
      field: iban
      format: electronic        # electronic (Standard), print (Vierergruppen) oder keep
      countryField: country     # neue Felder, leer: keines
      bankCodeField: blz
      # invalid: by-hand        # ungültige an diesen Transform statt auf den Fehler-Link
      # allowEmpty: true        # leere IBAN unverändert durchlassen
  - id: good
    type: csv.output
    with: { path: good.csv }
  - id: bad
    type: csv.output
    with: { path: bad.csv }     # Felder des Datensatzes plus _error, _errorField, _errorTransform
links:
  - { from: read, to: check }
  - { from: check, to: good }
  - { from: check, to: bad, type: error }
```

`runnel validate` und der Editor prüfen über `transform.init`, dass `field` im Eingang steht, die neuen Felder nicht
schon vorhanden sind und zu `invalid` ein Daten-Link führt; die neuen Felder sind danach bekannt (z. B. für
`fields` von `csv.output`). Ohne Fehler-Link und ohne `invalid` scheitert die Pipeline am ersten ungültigen Datensatz,
wie bei eingebauten Transforms. Durchsatz: etwa 320.000 Datensätze/s über das Protokoll (`make bench`).

### `iban.validate-bic` – BIC prüfen

Prüft die Form eines BIC (8 oder 11 Zeichen, Ländercode nach ISO 3166) – nicht, ob die Bank existiert.

```yaml
- id: bic
  type: iban.validate-bic
  with:
    bic: ${BIC}
```

Ergebnis: `valid`, `bic`, `bank` (4 Zeichen), `country`, `location`, `branch` (`XXX` für die Hauptstelle), `test`
(Test-BIC, zweites Zeichen des Ortes `0`), `error`.

### `iban.create` – IBAN bilden

Aus Ländercode und nationaler Kontonummer (BBAN) oder – für DE, AT, CH und LI – aus Bankleitzahl und Kontonummer (mit
Nullen links aufgefüllt):

```yaml
- id: create
  type: iban.create
  with:
    country: DE
    bankCode: "37040044"
    accountNumber: "532013000"
```

Ergebnis wie bei `iban.validate`. **Achtung (Deutschland):** Für einige Institute gelten
IBAN-Regeln der Deutschen Bundesbank;
das Plugin rechnet nach dem Standardverfahren. Für solche Konten die IBAN von der Bank verwenden.

## Entwickeln

| Zweck | Befehl |
|---|---|
| Bauen | `make build` → `bin/iban` |
| Tests (mit Race-Detector) | `make test` |
| Tests gegen Runnel | `make runnel-test RUNNEL_BIN=../runnel/bin/runnel` |
| Lint (golangci-lint v2.15.0) | `make lint` (ohne passendes golangci-lint: `make bin/tools/golangci-lint lint GOLANGCI_LINT=bin/tools/golangci-lint`) |
| Fuzzing | `make fuzz FUZZTIME=1m` |
| Benchmarks | `make bench` |
| Release-Archive | `make dist` → `dist/*.zip` mit erzeugten `THIRD_PARTY_NOTICES`, je Archiv eine SBOM (`*.cdx.json`, CycloneDX) und `dist/SHA256SUMS`; scheitert, wenn eine Abhängigkeit keine oder keine erlaubte Lizenz hat (MIT, BSD-2/3-Clause, Apache-2.0, ISC) |
| Lizenzhinweise ansehen | `make notices` |
| Release veröffentlichen | `version` in `plugin.yaml` und `main.go` erhöhen, mergen, dann Tag `v<version>` pushen oder den Workflow „Release“ auf `main` mit der Version starten; er prüft die Version, testet, baut die Archive mit Herkunftsnachweis und legt das GitHub-Release an |

Von Hand im Projekt testen: `make build`, dann `bin/iban` mit `plugin.yaml` nach `plugins/iban/` kopieren und
`runnel plugins trust`.

Das IBAN-Register steht in `internal/iban/registry.go`; zu jedem Land gibt es eine Beispiel-IBAN im Test. Ändert SWIFT
das Register, dort ergänzen. Die Version in `plugin.yaml` und `version` in `main.go` passen zum Tag (z. B. `v0.2.0`); ein
Test prüft das. Änderungen stehen in [CHANGELOG.md](CHANGELOG.md).
