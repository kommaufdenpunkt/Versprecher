# So funktioniert Insider – in einfachen Worten

Diese Seite erklärt das Projekt **ohne Fachchinesisch**. Sie ist für alle, die
verstehen wollen, was hier passiert, ohne selbst programmieren zu müssen.

## Worum geht's?

Insider ist ein kleines, **privates** Wörterbuch für lustige Versprecher und
Verhörer. Man sammelt sie in kleinen Gruppen, lacht gemeinsam und entdeckt sie
später wieder. Nichts wird öffentlich, solange nicht **alle Beteiligten**
zustimmen.

## Das Programm als „Büro" gedacht

Das Programm besteht aus vielen kleinen Bausteinen. Jeder macht **eine** Sache.
Stell es dir wie ein kleines Büro vor:

| Baustein (Ordner) | Aufgabe |
|---|---|
| `cmd/server` | Der **Anschalter** – startet alles. |
| `internal/config` | Liest die **Einstellungen** (z. B. Geheimnisse) aus der Umgebung. |
| `internal/db` | Die **Leitung zur Datenbank** (wo alles gespeichert wird). |
| `internal/auth` | Das **Türschloss**: Anmelden, Registrieren, Passwörter. |
| `internal/groups` | Die **Gruppen-Verwaltung**: Kreise gründen, einladen, beitreten. |
| `internal/middleware` | Der **Türsteher**: prüft Ausweise, bremst zu viele Anfragen. |
| `internal/httpx` | Der **Wegweiser**: welche Anfrage geht zu welchem Baustein. |
| `migrations` | Die **Baupläne** für die Datenbank-Tabellen. |
| `scripts` | Kleine **Helfer** zum Einrichten. |

Merksatz: Wenn etwas beim **Anmelden** klemmt, schaut man in `auth`. Wenn etwas
mit **Gruppen** klemmt, in `groups`. Mehr muss man sich nicht merken.

## Was kann die App heute schon?

### Anmelden & Sicherheit (Phase 1)
- **Nur auf Einladung**: Man kommt nur mit einem Einladungs-Link rein. Das ist
  der wichtigste Schutz vor Fremden und Bots.
- **Passwörter** werden **verschlüsselt** gespeichert (niemand kann sie lesen,
  auch wir nicht).
- **E-Mail bestätigen**: Erst nach Bestätigung kann man sich anmelden.
- **Bremse gegen Angreifer**: Wer zu schnell zu oft probiert (z. B. Passwörter
  raten), wird automatisch ausgebremst.

### Gruppen (Phase 2)
- Eine **Gruppe gründen** (man wird automatisch „Besitzer").
- **Leute einladen** – es entsteht ein einmaliger Einladungs-Link.
- Per Link **beitreten**: Neue Leute melden sich an und sind direkt in der
  Gruppe; wer schon ein Konto hat, tritt mit einem Klick bei.
- Eine Gruppe hat eine **Höchstzahl an Mitgliedern** (aktuell 30), damit sie
  klein und persönlich bleibt.

## Wie eine Einladung abläuft (Beispiel)

1. **Anna** gründet die Gruppe „Familie".
2. Anna erstellt eine **Einladung** und schickt **Ben** den Link.
3. Ben öffnet den Link und **registriert** sich → er ist sofort in der Gruppe.
4. Der Einladungs-Link ist **verbraucht** und funktioniert kein zweites Mal.

## Was kommt als Nächstes?

Die geplanten Schritte stehen in [ROADMAP.md](./ROADMAP.md). Als Nächstes:
Beiträge schreiben (die Versprecher selbst), der Helfer „Fidolin", Reaktionen
und Kommentare.

## Wenn du selbst mal reinschauen willst

- **Was ist geplant / fertig?** → [ROADMAP.md](./ROADMAP.md)
- **Was ist noch offen / zu entscheiden?** → [TODO.md](./TODO.md)
- **Die genaue Spezifikation** → [SPEC.md](./SPEC.md)
- **Wie starte ich es technisch?** → [README.md](./README.md)
