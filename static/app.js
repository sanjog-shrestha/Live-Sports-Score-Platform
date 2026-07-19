const scoresEl = document.getElementById("scores");
const statusEl = document.getElementById("status");
const standingsBody = document.getElementById("standingsBody");
const searchInput = document.getElementById("searchInput");
const favoritesOnlyCheckbox = document.getElementById("favoritesOnly");
const teamPicker = document.getElementById("teamPicker");
const addFavoriteBtn = document.getElementById("addFavoriteBtn");
const favoritesListEl = document.getElementById("favoritesList");
const teamPageEl = document.getElementById("teamPage");

const FAVORITES_KEY = "favoriteTeams";

function loadFavorites() {
    try {
        return new Set(JSON.parse(localStorage.getItem(FAVORITES_KEY) || "[]"));
    } catch {
        return new Set();
    }
}

function saveFavorites(favs) {
    localStorage.setItem(FAVORITES_KEY, JSON.stringify([...favs]));
}

let favorites = loadFavorites();
let allScores = [];
let currentStandings = [];

function statusClass(status) {
    if (status === "IN_PLAY" || status === "PAUSED") return "badge-live";
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

function matchesFavorites(s) {
    if (!favoritesOnlyCheckbox.checked) return true;
    return favorites.has(s.home) || favorites.has(s.away);
}

function applyFilters() {
    const filtered = filterScores(allScores, searchInput.value).filter(matchesFavorites);
    renderScores(filtered);
}

function toggleFavorite(teamName) {
    if (favorites.has(teamName)) {
        favorites.delete(teamName);
    } else {
        favorites.add(teamName);
    }
    saveFavorites(favorites);
    renderFavoritesList();
    applyFilters();
}

function buildTeamList(scores) {
    const set = new Set();
    for (const s of scores) {
        set.add(s.home);
        set.add(s.away);
    }
    return [...set].sort((a, b) => a.localeCompare(b));
}

function renderTeamPicker(teams) {
    teamPicker.innerHTML = '<option value="">Select a team…</option>';
    for (const team of teams) {
        const opt = document.createElement("option");
        opt.value = team;
        opt.textContent = team;
        teamPicker.appendChild(opt);
    }
}

function renderFavoritesList() {
    favoritesListEl.innerHTML = "";
    if (favorites.size === 0) {
        favoritesListEl.innerHTML = '<span class="empty">No favorites yet.</span>';
        return;
    }
    for (const team of [...favorites].sort((a, b) => a.localeCompare(b))) {
        const chip = document.createElement("span");
        chip.className = "favorite-chip";

        const label = document.createElement("span");
        label.textContent = team;

        const remove = document.createElement("button");
        remove.type = "button";
        remove.textContent = "×";
        remove.title = "Remove from favorites";
        remove.addEventListener("click", () => toggleFavorite(team));

        chip.appendChild(label);
        chip.appendChild(remove);
        favoritesListEl.appendChild(chip);
    }
}

function findStandingRow(teamName) {
    return currentStandings.find(r => r.team === teamName);
}

function renderTeamPage(teamName) {
    const row = findStandingRow(teamName);

    teamPageEl.innerHTML = "";
    teamPageEl.style.display = "block";

    const header = document.createElement("div");
    header.className = "team-page-header";

    const title = document.createElement("h3");
    title.textContent = teamName;

    const clearBtn = document.createElement("button");
    clearBtn.type = "button";
    clearBtn.textContent = "✕ Clear";
    clearBtn.addEventListener("click", clearTeamPage);

    header.appendChild(title);
    header.appendChild(clearBtn);
    teamPageEl.appendChild(header);

    if (row) {
        const stats = document.createElement("div");
        stats.className = "team-page-stats";
        stats.innerHTML = `
            <span>Position <strong>${row.position}</strong></span>
            <span>Played <strong>${row.played}</strong></span>
            <span>W-D-L <strong>${row.won}-${row.draw}-${row.lost}</strong></span>
            <span>GD <strong>${row.goalDiff}</strong></span>
            <span>Points <strong>${row.points}</strong></span>
            `;
        teamPageEl.appendChild(stats);
    } else {
        const note = document.createElement("div");
        note.className = "empty";
        note.textContent = "No standings data available for this team.";
        teamPageEl.appendChild(note);
    }
}

function clearTeamPage() {
    teamPageEl.style.display = "none";
    searchInput.value = "";
    applyFilters();
}

function showTeamPage(teamName) {
    searchInput.value = teamName;
    applyFilters();
    renderTeamPage(teamName);
}

function teamSpan(name) {
    const span = document.createElement("span");
    span.className = "team-name";

    const star = document.createElement("button");
    star.type = "button";
    star.className = "star-btn" + (favorites.has(name) ? " favorited" : "");
    star.textContent = favorites.has(name) ? "★" : "☆";
    star.title = favorites.has(name) ? "Remove from favorites" : "Add to favorites";
    star.addEventListener("click", (e) => {
        e.stopPropagation();
        toggleFavorite(name);
    });

    const label = document.createElement("span");
    label.className = "team-label";
    label.textContent = name;
    label.title = `View ${name}'s matches and standing`;
    label.addEventListener("click", (e) => {
        e.stopPropagation();
        showTeamPage(name);
    });

    span.appendChild(star);
    span.appendChild(label);
    return span;
}

function renderGoals(panel, goals, match) {
    panel.innerHTML = "";
    if (goals && goals.length > 0) {
        const list = document.createElement("ul");
        list.className = "goal-list";
        for (const g of goals) {
            const li = document.createElement("li");
            li.textContent = `${g.minute}' — ${g.scorer} (${g.team})`;
            list.appendChild(li);
        }
        panel.appendChild(list);
        return;
    }

    let message;
    if (match.id.startsWith("nba-")) {
        message = "Goal-by-play detail isn't available for this sport yet.";
    } else if (match.status === "SCHEDULED" || match.status === "TIMED") {
        message = "Match hasn't been played yet.";
    } else {
        message = "No goal data available for this match.";
    }
    panel.innerHTML = `<div class="empty">${message}</div>`;
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

        const home = teamSpan(s.home);
        const scorePill = document.createElement("span");
        scorePill.className = "score-pill";
        scorePill.textContent = `${s.homeScore} – ${s.awayScore}`;
        const away = teamSpan(s.away);

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
                goalsPanel.innerHTML = '<div class="empty">Loading goals…</div>';
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
            renderTeamPicker(buildTeamList(allScores));
            applyFilters();
        });
}

function loadStandings() {
    fetch("/api/standings")
        .then(res => res.json())
        .then(rows => {
            currentStandings = rows;
            renderStandings(rows);
            if (teamPageEl.style.display === "block" && searchInput.value) {
                renderTeamPage(searchInput.value);
            }
        });
}

renderFavoritesList();
loadScores();
loadStandings();

searchInput.addEventListener("input", applyFilters);
favoritesOnlyCheckbox.addEventListener("change", applyFilters);
addFavoriteBtn.addEventListener("click", () => {
    const team = teamPicker.value;
    if (!team) return;
    if (!favorites.has(team)) {
        toggleFavorite(team);
    }
    teamPicker.value = "";
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
        loadStandings();
    }
};
