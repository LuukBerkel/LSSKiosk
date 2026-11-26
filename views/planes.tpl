<div class="planes-page">
<div class="page-header">
    <h1>✈️ Vliegtuigen in de buurt</h1>
    <span class="badge badge-live"><span class="live-dot" aria-hidden="true"></span>Straal: 25 km • Bijgewerkt: {{.LastUpdated}}</span>
</div>

{{if .PlanesError}}
<div class="card">
    <p class="empty-state">{{.PlanesError}}</p>
</div>
{{else}}
<div class="grid-2 planes-grid">
    <div class="card highlight-card">
        <h3>Dichtstbijzijnde vliegtuig</h3>
        {{if .ClosestPlane}}
            <div class="aircraft-profile">
                <div class="aircraft-photo">
                    <img
                        class="aircraft-image"
                        src="{{if .ClosestPlane.ImageURL}}{{.ClosestPlane.ImageURL}}{{else}}/static/img/aircraft-placeholder.svg{{end}}"
                        data-placeholder="/static/img/aircraft-placeholder.svg"
                        alt="Vliegtuig {{.ClosestPlane.Callsign}}"
                        onerror="this.onerror=null;this.src=this.dataset.placeholder;">
                    {{if .ClosestPlane.ImageURL}}
                    <p class="photo-credit">
                        {{if .ClosestPlane.ImagePageURL}}<a href="{{.ClosestPlane.ImagePageURL}}" target="_blank" rel="noopener noreferrer">{{end}}
                        Foto: {{if .ClosestPlane.Photographer}}{{.ClosestPlane.Photographer}}{{else}}{{if .ClosestPlane.ImageSource}}{{.ClosestPlane.ImageSource}}{{else}}Planespotters.net{{end}}{{end}} · {{if .ClosestPlane.ImageSource}}{{.ClosestPlane.ImageSource}}{{else}}Planespotters.net{{end}}
                        {{if .ClosestPlane.ImagePageURL}}</a>{{end}}
                    </p>
                    {{end}}
                </div>
                <div class="aircraft-info">
                    <div class="big-value">{{.ClosestPlane.Callsign}}</div>
                    <p class="sub-value">{{.ClosestPlane.AircraftType}}{{if .ClosestPlane.Registration}} • {{.ClosestPlane.Registration}}{{end}}</p>
                </div>
            </div>
            <div class="stats-row">
                <div><strong>Hoogte:</strong> {{.ClosestPlane.Altitude}} m</div>
                <div><strong>Snelheid:</strong> {{.ClosestPlane.Speed}} km/u</div>
                <div><strong>Afstand:</strong> {{.ClosestPlane.Distance}} km</div>
                {{if .ClosestPlane.HasHeading}}<div><strong>Koers:</strong> {{.ClosestPlane.Heading}}°</div>{{end}}
                {{if .ClosestPlane.HasVertical}}<div><strong>Verticaal:</strong> {{.ClosestPlane.VerticalMove}} · {{.ClosestPlane.VerticalRate}} m/min</div>{{end}}
                {{if .ClosestPlane.Squawk}}<div><strong>Transpondercode:</strong> {{.ClosestPlane.Squawk}}</div>{{end}}
                {{if .ClosestPlane.Category}}<div><strong>ADS-B-categorie:</strong> {{.ClosestPlane.Category}}</div>{{end}}
                {{if .ClosestPlane.Hex}}<div><strong>ICAO-code:</strong> {{.ClosestPlane.Hex}}</div>{{end}}
                <div><strong>Positie:</strong> {{.ClosestPlane.Position}}</div>
            </div>
        {{else}}
            <p class="empty-state">Op dit moment zijn er geen vliegtuigen in de buurt.</p>
        {{end}}
    </div>

    <div class="card radar-card">
        <div class="radar-header">
            <div class="radar-title-group">
                <div class="radar-scope" aria-hidden="true">
                    <span class="radar-ring"></span>
                    <span class="radar-sweep"></span>
                    <span class="radar-blip radar-blip-1"></span>
                    <span class="radar-blip radar-blip-2"></span>
                </div>
                <div>
                    <h3>Radarverkeer</h3>
                    <p class="radar-subtitle">{{len .Planes}} vliegtuigen binnen 25 km (EHSE)</p>
                </div>
            </div>
        </div>
        <div class="table-scroll">
            <table class="kiosk-table">
                <thead>
                    <tr>
                        <th>Roepnaam</th>
                        <th>Hoogte (m)</th>
                        <th>Snelheid</th>
                        <th>Afstand</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Planes}}
                    <tr>
                        <td><strong>{{.Callsign}}</strong></td>
                        <td>{{.Altitude}}</td>
                        <td>{{.Speed}} km/u</td>
                        <td>{{.Distance}} km</td>
                    </tr>
                    {{else}}
                    <tr><td colspan="4">Geen vliegtuigen gedetecteerd.</td></tr>
                    {{end}}
                </tbody>
            </table>
        </div>
    </div>
</div>
{{end}}
</div>