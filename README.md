# Incident Replay & Load Simulator

Monorepo: backend + frontend + load (k6) + infra + agent tooling.

## Завантаження HAR (Chrome DevTools) для load-тесту

Можна використовувати запис мережевої активності з Chrome як сценарій для відтворення навантаження:

1. Відкрийте DevTools (F12) → вкладка **Network**.
2. Відтворіть потрібну активність (натисніть кнопки, перезавантажте сторінку тощо).
3. Клік правою кнопкою по списку запитів → **Save all as HAR with content** — збережіть `.har` файл.
4. У додатку перейдіть на **Scenarios** → секція **Upload HAR** → введіть Scenario ID і оберіть файл → **Upload HAR**.
5. Перейдіть на **Start Replay**: оберіть завантажений сценарій, вкажіть **Target Base URL** (куди слати запити), **Replay Mode** = **timestamp**, щоб зберегти інтервали між запитами, та RPS/Duration за потреби.
6. Запустіть replay і переглядайте результати в **Runs** та **Report**.

Деталі API та формат подій — у `docs/api.md` та `docs/events.md`.
