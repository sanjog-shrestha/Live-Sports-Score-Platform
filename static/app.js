const scoresEl = document.getElementById("scores");
const statusEl = document.getElementById("status");
const standingsBody = document.getElementById("standingsBody");
const searchInput = document.getElementById("searchInput");

let allScores = [];

function statusClass(status) {
    if (status === "IN_PLAY" || status === "PAUSED ") return "badge-live";
    if (status === "FINISHED" || status === "Final") return "badge-finished";
    if (status && status.includes("Qtr")) return "badge-live";
    return "badge-scheduled";
}

function filterScores(scores, query) {
    if (!query) return scores;
    const q = query.toLowerCase();
    return scores.filter(s =>
        s.home.toLowerCase().includes(q) || s.away.toLowerCase().includes(q)
    );
}

function renderGoals(panel, goals) {
    panel.innerHTML = "";
    if (!goals || goals.length === 0) {
        panel.innerHTML = '<div class="empty">No goal data available.</div>';
        return;
    }
    const list = document.createElement("ul");
    list.className = "goal-list";
    for (const g of goals) {
        const li = document.createElement("li");
        li.textContent = `${g.minute}' — ${g.scorer} (${g.team})`;
        list.appendChild(li);
    }
    panel.appendChild(list);
}


function renderScores(scores) {
    scoresEl.innerHTML = "";
    if (scores.length === 0) {
        scoresEl.innerHTML = '<div class="empty">No matches found.</div>';
        return;
    }
    for (const s of scores) {
        const row = document.createElement("div");
        row.className = "match";

        const main = document.createElement("div");
        main.className = "match-main";

        const comp = document.createElement("div");
        comp.className = "competition";
        comp.textContent = s.competition || "";

        const teamsRow = document.createElement("div");
        teamsRow.className = "teams-row";

        const home = document.createElement("span");
        home.className = "team-name";
        home.textContent = s.home;

        const scorePill = document.createElement("span");
        scorePill.className = "score-pill";
        scorePill.textContent = `${s.homeScore} - ${s.awayScore}`;

        const away = document.createElement("span");
        away.className = "team-name";
        away.textContent = s.away;

        teamsRow.appendChild(home);
        teamsRow.appendChild(scorePill);
        teamsRow.appendChild(away);

        const date = document.createElement("div");
        date.className = "status";
        date.textContent = s.date || "";

        main.appendChild(comp);
        main.appendChild(teamsRow);
        main.appendChild(date);

        const status = document.createElement("span");
        status.className = "badge " + statusClass(s.status);
        status.textContent = s.status || "";

        row.appendChild(main);
        row.appendChild(status);

        const goalsPanel = document.createElement("div");
        goalsPanel.className = "goals-panel";
        goalsPanel.style.display = "none";

        row.addEventListener("click", () => {
            const isOpen = goalsPanel.style.display === "block";
            if (isOpen) {
                goalsPanel.style.display = "none";
                return;
            }
            goalsPanel.style.display = "block";
            if (!goalsPanel.dataset.loaded) {
                goalsPanel.innerHTML = '<div class="empty">Loading goals..</div>';
                fetch(`/api/scores/${s.id}`)
                    .then(res => res.json())
                    .then(detail => {
                        goalsPanel.dataset.loaded = "true";
                        renderGoals(goalsPanel, detail.goals, s);
                    });
            }
        });

        scoresEl.appendChild(row);
        scoresEl.appendChild(goalsPanel);
    }
}

function renderStandings(rows) {
    standingsBody.innerHTML = "";
    for (const r of rows) {
        const tr = document.createElement("tr");
        const cells = [r.position, r.team, r.played, r.won, r.draw, r.lost, r.goalDiff, r.points];
        for (const val of cells) {
            const td = document.createElement("td");
            td.textContent = val;
            tr.appendChild(td);
        }
        standingsBody.appendChild(tr);
    }
}

function loadScores() {
    fetch("/api/scores")
        .then(res => res.json())
        .then(data => {
            allScores = data;
            renderScores(filterScores(allScores, searchInput.value));
        });
}

loadScores();
fetch("/api/standings").then(res => res.json()).then(renderStandings);

searchInput.addEventListener("input", () => {
    renderScores(filterScores(allScores, searchInput.value));
});

const ws = new WebSocket(`ws://${location.host}/ws`);

ws.onopen = () => { statusEl.textContent = "live"; };
ws.onclose = () => { statusEl.textContent = "disconnected"; };

ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.event === "score_added" || msg.event === "scores_refreshed") {
        loadScores();
    }
    if (msg.event === "standings_updated") {
        fetch("/api/standings").then(res => res.json()).then(renderStandings);
    }
};
