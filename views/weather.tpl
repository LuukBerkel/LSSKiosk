<div class="page-header">
    <h1>🌤️ Weer bij de scouting</h1>
    <span class="badge">Bijgewerkt: {{.LastUpdated}}</span>
</div>

<div class="grid-3">
    <div class="card center-card">
        <h3>Temperatuur</h3>
        <div class="big-value">{{.Weather.Temp}}°C</div>
        <p class="sub-value">Gevoelstemperatuur: {{.Weather.FeelsLike}}°C</p>
    </div>
    <div class="card center-card">
        <h3>Wind</h3>
        <div class="big-value">{{.Weather.WindSpeed}} <small>Bft</small></div>
        <p class="sub-value">Richting: {{.Weather.WindDir}} ({{.Weather.WindKmh}} km/u)</p>
    </div>

    <div class="card center-card">
        <h3>Neerslagkans</h3>
        <div class="big-value">{{.Weather.RainChance}}%</div>
        <p class="sub-value">{{.Weather.Condition}}</p>
    </div>
</div>

<div class="card mt-20">
    <h3>Komende uren</h3>
    <div class="forecast-row">
        {{range .Forecast}}
        <div class="forecast-item">
            <div class="time">{{.Hour}}</div>
            <div class="icon">{{.Icon}}</div>
            <div class="temp"><strong>{{.Temp}}°C</strong></div>
            <div class="rain">💧 {{.Rain}}%</div>
        </div>
        {{end}}
    </div>
</div>