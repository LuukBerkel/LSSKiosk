<!doctype html>
<html lang="nl">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="theme-color" content="#0f172a">
    <title>Scoutingdashboard</title>
    <link rel="stylesheet" href="/static/css/kiosk.css">
</head>
<body data-active-slide="{{.ActiveSlide}}">
    <div class="sky-backdrop" aria-hidden="true"></div>

    <header class="kiosk-header">
        <div class="brand">
            <img class="brand-logo" src="/static/img/logo.png" alt="Luchtscouts Seppe logo">
            <span>Scoutingdashboard</span>
        </div>
        <div class="clock" aria-label="Huidige tijd in Nederland">{{.CurrentTime}}</div>
    </header>

    <main class="kiosk-main">
        <div class="slide-viewport" aria-live="polite">
            <div class="slide-track">
                <section class="dashboard-slide" id="slide-energy" aria-label="Energie">
                    <p class="slide-loading">Energiegegevens laden…</p>
                </section>
                <section class="dashboard-slide" id="slide-planes" aria-label="Vliegtuigen">
                    <p class="slide-loading">Vliegtuiggegevens laden…</p>
                </section>
                <section class="dashboard-slide" id="slide-weather" aria-label="Weer">
                    <p class="slide-loading">Weersinformatie laden…</p>
                </section>
            </div>
        </div>
    </main>

    <nav class="kiosk-nav" aria-label="Dashboardpagina's">
        <button class="nav-dot" type="button" aria-label="Toon energie" title="Energie"></button>
        <button class="nav-dot" type="button" aria-label="Toon vliegtuigen" title="Vliegtuigen"></button>
        <button class="nav-dot" type="button" aria-label="Toon het weer" title="Weer"></button>
    </nav>
    <script src="/static/js/dashboard.js"></script>
</body>
</html>
