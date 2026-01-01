const {WatchClubServiceClient} = require('./api/v1_grpc_web_pb.js');
const {
    CreateUserRequest,
    GetUserRequest,
    CreateClubRequest,
    JoinClubRequest,
    AddPickRequest,
    DeletePickRequest,
    GetClubRequest,
    StartClubRequest,
    GetScheduledPicksRequest,
    SendLoginEmailRequest,
    GetClubCalendarRequest
} = require('./api/v1_pb.js');

const {Timestamp} = require('google-protobuf/google/protobuf/timestamp_pb.js');

const client = new WatchClubServiceClient(window.location.origin, null, null);

// ===== STATE MANAGEMENT =====
const state = {
    currentUser: null,
    clubs: [], // clubs the user is a member of

    loadUser() {
        const stored = localStorage.getItem('watchclub_user');
        if (stored) {
            this.currentUser = JSON.parse(stored);
        }
    },

    saveUser(user) {
        this.currentUser = {
            id: user.getId(),
            name: user.getName(),
            email: user.getEmail()
        };
        localStorage.setItem('watchclub_user', JSON.stringify(this.currentUser));
        this.loadClubs();
        router.updateNav();
    },

    clearUser() {
        this.currentUser = null;
        this.clubs = [];
        localStorage.removeItem('watchclub_user');
        localStorage.removeItem('watchclub_clubs');
        router.updateNav();
    },

    loadClubs() {
        const stored = localStorage.getItem('watchclub_clubs');
        if (stored) {
            this.clubs = JSON.parse(stored);
        }
    },

    addClub(club) {
        const clubData = {
            id: club.getId(),
            name: club.getName(),
            startDate: club.getStartDate() ? club.getStartDate().getSeconds() : null,
            started: club.getStarted(),
            maxPicksPerMember: club.getMaxPicksPerMember() || 1,
            scheduleIntervalQuantity: club.getScheduleIntervalQuantity() || 1,
            scheduleIntervalUnit: club.getScheduleIntervalUnit() || 2
        };

        // Remove if exists and add to front
        this.clubs = this.clubs.filter(c => c.id !== clubData.id);
        this.clubs.unshift(clubData);
        localStorage.setItem('watchclub_clubs', JSON.stringify(this.clubs));
    }
};

// ===== HELPER FUNCTIONS =====
function formatDate(timestamp) {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp.getSeconds() * 1000);
    return date.toLocaleDateString();
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ===== ROUTER =====
const router = {
    routes: {},

    init() {
        window.addEventListener('hashchange', () => this.handleRoute());
        this.handleRoute();
    },

    register(path, handler) {
        this.routes[path] = handler;
    },

    navigate(path) {
        window.location.hash = path;
    },

    handleRoute() {
        let hash = window.location.hash.slice(1) || '/';

        // Parse route with parameters
        let matchedRoute = null;
        let params = {};

        for (let route in this.routes) {
            const pattern = route.replace(/:[^\s/]+/g, '([^/]+)');
            const regex = new RegExp(`^${pattern}$`);
            const match = hash.match(regex);

            if (match) {
                matchedRoute = route;
                const paramNames = route.match(/:[^\s/]+/g) || [];
                paramNames.forEach((name, i) => {
                    params[name.slice(1)] = match[i + 1];
                });
                break;
            }
        }

        if (matchedRoute) {
            this.routes[matchedRoute](params);
        } else {
            this.routes['/']();
        }

        this.updateNav();
    },

    updateNav() {
        const navLinks = document.getElementById('navLinks');
        if (!navLinks) return;

        if (state.currentUser) {
            navLinks.innerHTML = `
                <a href="#/my-clubs">My Clubs</a>
                <a href="#/profile">Profile</a>
                <a href="#/" onclick="logout(); return false;">Logout</a>
            `;
        } else {
            navLinks.innerHTML = '';
        }
    }
};

// ===== PAGE VIEWS =====

// Home Page - Create or Join Club
function renderHomePage() {
    const content = document.getElementById('app-content');
    content.innerHTML = `
        <div class="home-page">
            ${!state.currentUser ? `
                <header class="page-header">
                    <h1>👋 Hello!</h1>
                    <p>WatchClub helps you watch stuff together. Create a club, invite your friends, add picks, and start watching!</p>
                </header>
            ` : ''}

            ${!state.currentUser ? `
                <div class="card">
                    <h2>First, create your account</h2>
                    <div class="form-group">
                        <input type="text" id="userName" placeholder="Your name">
                        <input type="text" id="userEmail" placeholder="Your email">
                        <button onclick="createUserAccount()">Create Account</button>
                    </div>
                    <p style="margin-top: 1rem; text-align: center;">
                        <a href="#/login" class="btn-link">Already have an account? Log in</a>
                    </p>
                    <div id="userError" class="error-message"></div>
                </div>
            ` : `
                ${state.clubs.length > 0 ? `
                    <div class="card">
                        <h2>Your Clubs</h2>
                        <div class="club-list">
                            ${state.clubs.map(club => `
                                <a href="#/club/${club.id}" class="club-item">
                                    <strong>${escapeHtml(club.name)}</strong>
                                    <span class="club-status ${club.started ? 'started' : 'pending'}">
                                        ${club.started ? '✓ Started' : 'Pending'}
                                    </span>
                                </a>
                            `).join('')}
                        </div>
                        <a href="#/my-clubs" class="btn-link">View all clubs →</a>
                    </div>
                ` : ''}

                <div class="actions-grid">
                    <div class="card action-card">
                        <h2>Create a Club</h2>
                        <p>Start a new watch club and invite your friends</p>
                        <div class="form-group">
                            <label>Club name</label>
                            <input type="text" id="clubName" placeholder="e.g., 2026 Movie Club">
                            <label>Start date</label>
                            <input type="date" id="clubStartDate">
                            <label>Max picks per member</label>
                            <input type="number" id="clubMaxPicks" min="1" value="1">
                            <label>Schedule interval</label>
                            <div class="interval-inputs" style="display: flex; gap: 0.5rem;">
                                <input type="number" id="scheduleQuantity" min="1" value="1" placeholder="1" style="flex: 1;">
                                <select id="scheduleUnit" style="flex: 1;">
                                    <option value="1">Days</option>
                                    <option value="2" selected>Weeks</option>
                                    <option value="3">Months</option>
                                </select>
                            </div>
                            <button onclick="createClub()">Create Club</button>
                        </div>
                        <div id="createClubError" class="error-message"></div>
                    </div>

                    <div class="card action-card">
                        <h2>Join a Club</h2>
                        <p>Enter a club code to join</p>
                        <div class="form-group">
                            <input type="text" id="joinClubCode" placeholder="Club code">
                            <button onclick="joinClubByCode()">Join Club</button>
                        </div>
                        <div id="joinClubError" class="error-message"></div>
                    </div>
                </div>
            `}
        </div>
    `;
}

// Join Club Page
function renderJoinPage(params) {
    const content = document.getElementById('app-content');
    const clubId = params.clubId;

    content.innerHTML = `
        <div class="join-page">
            <div class="card">
                <h1>Join Club</h1>
                <div id="clubInfo">Loading club info...</div>
                ${!state.currentUser ? `
                    <div class="form-group">
                        <label>What's your name?</label>
                        <input type="text" id="userName" placeholder="Enter your name">
                        <label>What's your email?</label>
                        <input type="text" id="userEmail" placeholder="Enter your email">
                    </div>
                ` : `
                    <p>Joining as <strong>${escapeHtml(state.currentUser.name)}</strong></p>
                `}
                <button onclick="joinClubAction('${clubId}')">Join Club</button>
                <div id="joinError" class="error-message"></div>
            </div>
        </div>
    `;

    // Load club info
    const request = new GetClubRequest();
    request.setClubId(clubId);

    client.getClub(request, {}, (err, response) => {
        const clubInfo = document.getElementById('clubInfo');
        if (err) {
            clubInfo.innerHTML = `<p class="error-message">Error loading club: ${err.message}</p>`;
            return;
        }

        const club = response.getClub();
        const members = response.getMembersList();

        clubInfo.innerHTML = `
            <h2>${escapeHtml(club.getName())}</h2>
            <p><strong>Start Date:</strong> ${formatDate(club.getStartDate())}</p>
            <p><strong>Members:</strong> ${members.length}</p>
        `;
    });
}

// My Clubs Page
function renderMyClubsPage() {
    const content = document.getElementById('app-content');

    if (!state.currentUser) {
        router.navigate('/');
        return;
    }

    content.innerHTML = `
        <div class="my-clubs-page">
            <div class="page-header">
                <h1>My Clubs</h1>
                <a href="#/" class="btn-secondary">+ Create New Club</a>
            </div>

            ${state.clubs.length === 0 ? `
                <div class="card empty-state">
                    <p>You haven't joined any clubs yet.</p>
                    <a href="#/" class="btn">Create or Join a Club</a>
                </div>
            ` : `
                <div class="clubs-grid">
                    ${state.clubs.map(club => `
                        <div class="card club-card">
                            <h3>${escapeHtml(club.name)}</h3>
                            <p class="club-status ${club.started ? 'started' : 'pending'}">
                                ${club.started ? '✓ Started' : 'Pending'}
                            </p>
                            <a href="#/club/${club.id}" class="btn">View Details</a>
                        </div>
                    `).join('')}
                </div>
            `}
        </div>
    `;
}

// Profile Page
function renderProfilePage() {
    const content = document.getElementById('app-content');

    if (!state.currentUser) {
        router.navigate('/');
        return;
    }

    content.innerHTML = `
        <div class="profile-page">
            <div class="page-header">
                <h1>Profile</h1>
            </div>

            <div class="card">
                <div class="profile-info">
                    <div class="profile-field">
                        <label>Name</label>
                        <p>${escapeHtml(state.currentUser.name)}</p>
                    </div>
                    <div class="profile-field">
                        <label>Email</label>
                        <p>${escapeHtml(state.currentUser.email)}</p>
                    </div>
                    <div class="profile-field">
                        <label>User ID</label>
                        <p class="user-id">${escapeHtml(state.currentUser.id)}</p>
                    </div>
                </div>
            </div>
        </div>
    `;
}

// Club Detail Page
function renderClubDetailPage(params) {
    const content = document.getElementById('app-content');
    const clubId = params.clubId;

    if (!state.currentUser) {
        router.navigate('/');
        return;
    }

    content.innerHTML = `
        <div class="club-detail-page">
            <div id="clubContent">Loading...</div>
        </div>
    `;

    const request = new GetClubRequest();
    request.setClubId(clubId);

    client.getClub(request, {}, (err, response) => {
        const clubContent = document.getElementById('clubContent');
        if (err) {
            clubContent.innerHTML = `
                <div class="card">
                    <p class="error-message">Error loading club: ${err.message}</p>
                    <a href="#/my-clubs" class="btn">Back to My Clubs</a>
                </div>
            `;
            return;
        }

        const club = response.getClub();
        const members = response.getMembersList();
        const picks = response.getPicksList();

        // Check if user is a member
        const isMember = members.some(m => m.getId() === state.currentUser.id);

        if (!isMember) {
            clubContent.innerHTML = `
                <div class="card">
                    <p class="error-message">You are not a member of this club.</p>
                    <a href="#/my-clubs" class="btn">Back to My Clubs</a>
                </div>
            `;
            return;
        }

        state.addClub(club);

        const userPicks = picks.filter(p => p.getUserId() === state.currentUser.id);
        const maxPicks = club.getMaxPicksPerMember();
        const canAddMore = maxPicks === 0 || userPicks.length < maxPicks;
        const shareUrl = `${window.location.origin}${window.location.pathname}#/club/${clubId}/join`;

        clubContent.innerHTML = `
            <div class="page-header">
                <h1>${escapeHtml(club.getName())}</h1>
                <div class="club-meta">
                    <span>Start Date: ${formatDate(club.getStartDate())}</span>
                    <span>Schedule: Every ${club.getScheduleIntervalQuantity()} ${getUnitName(club.getScheduleIntervalUnit(), club.getScheduleIntervalQuantity())}</span>
                    <span class="club-status ${club.getStarted() ? 'started' : 'pending'}">
                        ${club.getStarted() ? '✓ Started' : 'Pending'}
                    </span>
                </div>
            </div>

            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1.5rem; margin-bottom: 1.5rem;">
                <div class="card">
                    <h3>Invite Friends</h3>
                    <p>Share this link with your friends:</p>
                    <div class="share-link">
                        <input type="text" value="${shareUrl}" readonly onclick="this.select()">
                        <button onclick="copyShareLink('${shareUrl}')">Copy</button>
                    </div>
                    <div id="copySuccess" class="success-message" style="display: none;">Link copied!</div>
                </div>

                <div class="card">
                    <h3>Members (${members.length})</h3>
                    <div class="member-list">
                        ${members.map(m => {
                            const memberPickCount = picks.filter(p => p.getUserId() === m.getId()).length;
                            const maxPicksDisplay = maxPicks === 0 ? '∞' : maxPicks;
                            return `
                                <div class="member-item">
                                    <span>${escapeHtml(m.getName())}</span>
                                    <span class="badge ${memberPickCount > 0 ? 'success' : 'pending'}">
                                        ${memberPickCount}/${maxPicksDisplay} picks
                                    </span>
                                </div>
                            `;
                        }).join('')}
                    </div>
                </div>
            </div>

            ${!club.getStarted() ? `
                <div class="card">
                    <h3>Your Picks (${userPicks.length}/${maxPicks === 0 ? '∞' : maxPicks})</h3>

                    ${userPicks.length > 0 ? `
                        <div class="user-picks-list">
                            ${userPicks.map(pick => `
                                <div class="pick-item user-pick">
                                    <div class="pick-content">
                                        <strong>${escapeHtml(pick.getTitle())}</strong>
                                        ${pick.getYear() ? `(${pick.getYear()})` : ''}
                                        ${pick.getNotes() ? `<p class="pick-notes">${escapeHtml(pick.getNotes())}</p>` : ''}
                                    </div>
                                    <button class="btn-danger btn-small" onclick="deletePickAction('${clubId}', '${pick.getId()}')">
                                        Delete
                                    </button>
                                </div>
                            `).join('')}
                        </div>
                    ` : '<p>You haven\'t added any picks yet.</p>'}

                    ${canAddMore ? `
                        <a href="#/club/${clubId}/add-pick" class="btn">
                            ${userPicks.length === 0 ? 'Add A Pick' : 'Add Another Pick'}
                        </a>
                    ` : `
                        <p class="info-message">You've reached the maximum number of picks (${maxPicks}).</p>
                    `}

                    ${picks.length > 0 ? `
                        <h4 style="margin-top: 2rem;">All Picks (${picks.length})</h4>
                        <div class="pick-list">
                            ${picks.map(p => {
                                const pickMember = members.find(m => m.getId() === p.getUserId());
                                return `
                                    <div class="pick-item">
                                        <strong>${escapeHtml(p.getTitle())}</strong> ${p.getYear() ? `(${p.getYear()})` : ''}
                                        <span class="pick-author">by ${escapeHtml(pickMember ? pickMember.getName() : 'Unknown')}</span>
                                        ${p.getNotes() ? `<p class="pick-notes">${escapeHtml(p.getNotes())}</p>` : ''}
                                    </div>
                                `;
                            }).join('')}
                        </div>
                    ` : ''}
                </div>
            ` : ''}

            ${!club.getStarted() && picks.length > 0 ? `
                <div class="card">
                    <h3>Ready to Start!</h3>
                    <p>The club has ${picks.length} pick${picks.length !== 1 ? 's' : ''}. Click below to shuffle and generate the schedule.</p>
                    <button onclick="startClubAction('${clubId}')" class="btn primary">Start Club & Shuffle</button>
                    <div id="startError" class="error-message"></div>
                </div>
            ` : !club.getStarted() ? `
                <div class="card">
                    <h3>Waiting for Picks</h3>
                    <p>The club needs at least one pick before it can be started.</p>
                </div>
            ` : club.getStarted() ? `
                <div class="card">
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
                        <h3 style="margin: 0;">Schedule</h3>
                        <button onclick="downloadCalendar('${clubId}')" class="btn-secondary">Download</button>
                    </div>
                    <div id="scheduleContent">Loading schedule...</div>
                </div>
            ` : ''}
        `;

        if (club.getStarted()) {
            loadSchedule(clubId, club, members);
        }
    });
}

// Add Pick Page
function renderAddPickPage(params) {
    const content = document.getElementById('app-content');
    const clubId = params.clubId;

    if (!state.currentUser) {
        router.navigate('/');
        return;
    }

    content.innerHTML = `
        <div class="add-pick-page">
            <div class="card">
                <h1>Add A Pick</h1>
                <div class="form-group">
                    <label>Title *</label>
                    <input type="text" id="title" placeholder="e.g., The Shawshank Redemption">

                    <label>Year</label>
                    <input type="number" id="year" placeholder="e.g., 1994">

                    <label>Link to watch</label>
                    <input type="url" id="link" placeholder="e.g., https://netflix.com/...">

                    <label>Why did you pick this?</label>
                    <textarea id="notes" rows="3" placeholder="Optional notes..."></textarea>
                </div>

                <div class="button-group">
                    <button onclick="addPickAction('${clubId}')">Add Pick</button>
                    <a href="#/club/${clubId}" class="btn btn-secondary">Cancel</a>
                </div>

                <div id="addPickError" class="error-message"></div>
            </div>
        </div>
    `;
}

// Pick Detail Page
function renderPickDetailPage(params) {
    const content = document.getElementById('app-content');
    const { clubId, pickId } = params;

    if (!state.currentUser) {
        router.navigate('/');
        return;
    }

    content.innerHTML = `
        <div class="pick-detail-page">
            <div class="card">
                <div id="pickDetailContent">Loading...</div>
            </div>
        </div>
    `;

    const request = new GetClubRequest();
    request.setClubId(clubId);

    client.getClub(request, {}, (err, response) => {
        const pickDetailContent = document.getElementById('pickDetailContent');
        if (err) {
            pickDetailContent.innerHTML = `
                <p class="error-message">Error loading pick details: ${err.message}</p>
                <a href="#/club/${clubId}" class="btn">Back to Club</a>
            `;
            return;
        }

        const club = response.getClub();
        const members = response.getMembersList();
        const picks = response.getPicksList();
        const pick = picks.find(p => p.getId() === pickId);

        if (!pick) {
            pickDetailContent.innerHTML = `
                <p class="error-message">Pick not found</p>
                <a href="#/club/${clubId}" class="btn">Back to Club</a>
            `;
            return;
        }

        const member = members.find(m => m.getId() === pick.getUserId());

        pickDetailContent.innerHTML = `
            <h1>${escapeHtml(pick.getTitle())}${pick.getYear() ? ` (${pick.getYear()})` : ''}</h1>
            <p class="pick-meta">Picked by ${escapeHtml(member ? member.getName() : 'Unknown')}</p>

            ${pick.getLink() ? `
                <div class="pick-link-section" style="margin-top: 1.5rem;">
                    <a href="${escapeHtml(pick.getLink())}" target="_blank" rel="noopener noreferrer" class="btn primary">
                        Watch Now →
                    </a>
                </div>
            ` : ''}

            ${pick.getNotes() ? `
                <div class="pick-notes-section" style="margin-top: 1.5rem;">
                    <h3>Notes</h3>
                    <p class="pick-notes-detail">${escapeHtml(pick.getNotes())}</p>
                </div>
            ` : !pick.getLink() ? '<p class="info-message" style="margin-top: 1.5rem;">No additional details added for this pick.</p>' : ''}

            <div class="button-group" style="margin-top: 2rem;">
                <a href="#/club/${clubId}" class="btn">Back to Club</a>
            </div>
        `;
    });
}

// ===== ACTIONS =====

function createUserAccount() {
    const name = document.getElementById('userName').value.trim();
    const email = document.getElementById('userEmail').value.trim();
    const errorEl = document.getElementById('userError');

    if (!name) {
        errorEl.textContent = 'Please enter your name';
        errorEl.style.display = 'block';
        return;
    }

    if (!email) {
        errorEl.textContent = 'Please enter your email';
        errorEl.style.display = 'block';
        return;
    }

    const request = new CreateUserRequest();
    request.setName(name);
    request.setEmail(email);

    client.createUser(request, {}, (err, response) => {
        if (err) {
            errorEl.textContent = `Error: ${err.message}`;
            errorEl.style.display = 'block';
            return;
        }

        state.saveUser(response.getUser());
        renderHomePage();
    });
}

function createClub() {
    const name = document.getElementById('clubName').value.trim();
    const startDateStr = document.getElementById('clubStartDate').value;
    const maxPicks = parseInt(document.getElementById('clubMaxPicks').value) || 1;
    const scheduleQty = parseInt(document.getElementById('scheduleQuantity').value) || 1;
    const scheduleUnit = parseInt(document.getElementById('scheduleUnit').value) || 2;
    const errorEl = document.getElementById('createClubError');

    if (!name || !startDateStr) {
        errorEl.textContent = 'Please fill in all fields';
        errorEl.style.display = 'block';
        return;
    }

    if (maxPicks < 1) {
        errorEl.textContent = 'Max picks must be at least 1';
        errorEl.style.display = 'block';
        return;
    }

    if (scheduleQty < 1) {
        errorEl.textContent = 'Schedule interval must be at least 1';
        errorEl.style.display = 'block';
        return;
    }

    const request = new CreateClubRequest();
    request.setName(name);
    request.setMaxPicksPerMember(maxPicks);
    request.setScheduleIntervalQuantity(scheduleQty);
    request.setScheduleIntervalUnit(scheduleUnit);

    // Parse date as local time, not UTC
    const [year, month, day] = startDateStr.split('-').map(Number);
    const startDate = new Date(year, month - 1, day); // month is 0-indexed
    const timestamp = new Timestamp();
    timestamp.setSeconds(Math.floor(startDate.getTime() / 1000));
    request.setStartDate(timestamp);

    client.createClub(request, {}, (err, response) => {
        if (err) {
            errorEl.textContent = `Error: ${err.message}`;
            errorEl.style.display = 'block';
            return;
        }

        const club = response.getClub();

        // Auto-join the creator
        const joinRequest = new JoinClubRequest();
        joinRequest.setClubId(club.getId());
        joinRequest.setUserId(state.currentUser.id);

        client.joinClub(joinRequest, {}, (err) => {
            if (err) {
                errorEl.textContent = `Club created but failed to join: ${err.message}`;
                errorEl.style.display = 'block';
                return;
            }

            state.addClub(club);
            router.navigate(`/club/${club.getId()}`);
        });
    });
}

function joinClubByCode() {
    const code = document.getElementById('joinClubCode').value.trim();
    if (code) {
        router.navigate(`/club/${code}/join`);
    }
}

function joinClubAction(clubId) {
    const errorEl = document.getElementById('joinError');

    // Create user if needed
    if (!state.currentUser) {
        const name = document.getElementById('userName').value.trim();
        const email = document.getElementById('userEmail').value.trim();

        if (!name) {
            errorEl.textContent = 'Please enter your name';
            errorEl.style.display = 'block';
            return;
        }

        if (!email) {
            errorEl.textContent = 'Please enter your email';
            errorEl.style.display = 'block';
            return;
        }

        const userRequest = new CreateUserRequest();
        userRequest.setName(name);
        userRequest.setEmail(email);

        client.createUser(userRequest, {}, (err, response) => {
            if (err) {
                errorEl.textContent = `Error: ${err.message}`;
                errorEl.style.display = 'block';
                return;
            }

            state.saveUser(response.getUser());
            performJoin(clubId, errorEl);
        });
    } else {
        performJoin(clubId, errorEl);
    }
}

function performJoin(clubId, errorEl) {
    const request = new JoinClubRequest();
    request.setClubId(clubId);
    request.setUserId(state.currentUser.id);

    client.joinClub(request, {}, (err, response) => {
        if (err) {
            errorEl.textContent = `Error: ${err.message}`;
            errorEl.style.display = 'block';
            return;
        }

        state.addClub(response.getClub());
        router.navigate(`/club/${clubId}`);
    });
}

function addPickAction(clubId) {
    const title = document.getElementById('title').value.trim();
    const year = parseInt(document.getElementById('year').value);
    const link = document.getElementById('link').value.trim();
    const notes = document.getElementById('notes').value.trim();
    const errorEl = document.getElementById('addPickError');

    if (!title) {
        errorEl.textContent = 'Please enter a title';
        errorEl.style.display = 'block';
        return;
    }

    const request = new AddPickRequest();
    request.setClubId(clubId);
    request.setUserId(state.currentUser.id);
    request.setTitle(title);
    if (year) request.setYear(year);
    if (link) request.setLink(link);
    if (notes) request.setNotes(notes);

    client.addPick(request, {}, (err) => {
        if (err) {
            errorEl.textContent = `Error: ${err.message}`;
            errorEl.style.display = 'block';
            return;
        }

        router.navigate(`/club/${clubId}`);
    });
}

function startClubAction(clubId) {
    const errorEl = document.getElementById('startError');
    const request = new StartClubRequest();
    request.setClubId(clubId);

    client.startClub(request, {}, (err, response) => {
        if (err) {
            errorEl.textContent = `Error: ${err.message}`;
            errorEl.style.display = 'block';
            return;
        }

        state.addClub(response.getClub());
        // Re-render the current page to show updated state
        renderClubDetailPage({clubId});
    });
}

function deletePickAction(clubId, pickId) {
    if (!confirm('Are you sure you want to delete this pick?')) {
        return;
    }

    const request = new DeletePickRequest();
    request.setPickId(pickId);
    request.setUserId(state.currentUser.id);

    client.deletePick(request, {}, (err) => {
        if (err) {
            alert(`Error deleting pick: ${err.message}`);
            return;
        }

        // Refresh the club detail page
        renderClubDetailPage({clubId});
    });
}

function downloadCalendar(clubId) {
    const request = new GetClubCalendarRequest();
    request.setClubId(clubId);

    client.getClubCalendar(request, {}, (err, response) => {
        if (err) {
            alert(`Error generating calendar: ${err.message}`);
            return;
        }

        const icsData = response.getIcsData();
        const blob = new Blob([icsData], { type: 'text/calendar;charset=utf-8' });
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'watchclub-schedule.ics';
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
    });
}

function loadSchedule(clubId, club, members) {
    const request = new GetScheduledPicksRequest();
    request.setClubId(clubId);

    client.getScheduledPicks(request, {}, (err, response) => {
        const scheduleContent = document.getElementById('scheduleContent');
        if (err) {
            scheduleContent.innerHTML = `<p class="error-message">Error loading schedule: ${err.message}</p>`;
            return;
        }

        const assignments = response.getAssignmentsList();

        scheduleContent.innerHTML = `
            <div class="schedule-list">
                ${assignments.map(a => {
                    const pick = a.getPick();
                    const member = members.find(m => m.getId() === pick.getUserId());
                    return `
                        <a href="#/club/${clubId}/pick/${pick.getId()}" class="schedule-item">
                            <div class="week-number">#${a.getSequenceNumber()}</div>
                            <div class="schedule-details">
                                <strong>${escapeHtml(pick.getTitle())}</strong> ${pick.getYear() ? `(${pick.getYear()})` : ''}
                                <span class="pick-author">by ${escapeHtml(member ? member.getName() : 'Unknown')}</span>
                                <div class="schedule-date">${formatDate(a.getStartDate())}</div>
                            </div>
                        </a>
                    `;
                }).join('')}
            </div>
        `;
    });
}

// Helper function to get unit name with proper pluralization
function getUnitName(unit, quantity) {
    const plural = quantity !== 1;
    switch(unit) {
        case 1: // DAYS
            return plural ? 'days' : 'day';
        case 2: // WEEKS
            return plural ? 'weeks' : 'week';
        case 3: // MONTHS
            return plural ? 'months' : 'month';
        default:
            return 'periods';
    }
}

function copyShareLink(url) {
    navigator.clipboard.writeText(url).then(() => {
        const success = document.getElementById('copySuccess');
        success.style.display = 'block';
        setTimeout(() => success.style.display = 'none', 2000);
    });
}

function logout() {
    state.clearUser();
    router.navigate('/');
    renderHomePage(); // Force re-render after logout
}

// Make functions globally available
window.createUserAccount = createUserAccount;
window.createClub = createClub;
window.joinClubByCode = joinClubByCode;
window.joinClubAction = joinClubAction;
window.addPickAction = addPickAction;
window.startClubAction = startClubAction;
window.deletePickAction = deletePickAction;
window.downloadCalendar = downloadCalendar;
window.copyShareLink = copyShareLink;
window.logout = logout;

// ===== INITIALIZE =====
state.loadUser();
state.loadClubs();

// Login Page
function renderLoginPage() {
    const content = document.getElementById('app-content');
    content.innerHTML = `
        <div class="login-page">
            <div class="card">
                <h1>Log in to your account</h1>
                <p>Enter your email address and we'll send you a link to log back in.</p>
                <div class="form-group">
                    <input type="text" id="loginEmail" placeholder="Your email">
                    <button onclick="sendLoginEmail()">Send Login Link</button>
                </div>
                <div id="loginResult" class="result"></div>
                <p style="margin-top: 1rem; text-align: center;">
                    <a href="#/" class="btn-link">← Back to home</a>
                </p>
            </div>
        </div>
    `;
}

// Auto-Login Page
function renderAutoLoginPage(params) {
    const userId = params.userId;
    const content = document.getElementById('app-content');

    content.innerHTML = `
        <div class="card">
            <h1>Logging you in...</h1>
            <p>Please wait while we verify your account.</p>
        </div>
    `;

    const request = new GetUserRequest();
    request.setUserId(userId);

    client.getUser(request, {}, (err, response) => {
        if (err) {
            content.innerHTML = `
                <div class="card">
                    <h1>Invalid Login Link</h1>
                    <p class="error-message">This login link is invalid or expired.</p>
                    <a href="#/" class="btn">Go to Home</a>
                </div>
            `;
            return;
        }

        state.saveUser(response.getUser());
        router.navigate('/my-clubs');
    });
}

function sendLoginEmail() {
    const email = document.getElementById('loginEmail').value.trim();
    const resultEl = document.getElementById('loginResult');

    if (!email) {
        resultEl.innerHTML = 'Please enter your email';
        resultEl.className = 'result error';
        resultEl.style.display = 'block';
        return;
    }

    const request = new SendLoginEmailRequest();
    request.setEmail(email);

    client.sendLoginEmail(request, {}, (err, response) => {
        if (err) {
            resultEl.innerHTML = `Error: ${err.message}`;
            resultEl.className = 'result error';
            resultEl.style.display = 'block';
            return;
        }

        resultEl.innerHTML = `
            <strong>Check your email!</strong><br>
            ${response.getMessage()}<br><br>
        `;
        resultEl.className = 'result success';
        resultEl.style.display = 'block';
    });
}

// Make globally available
window.sendLoginEmail = sendLoginEmail;

// Register routes
router.register('/', renderHomePage);
router.register('/login', renderLoginPage);
router.register('/login/:userId', renderAutoLoginPage);
router.register('/club/:clubId/join', renderJoinPage);
router.register('/my-clubs', renderMyClubsPage);
router.register('/profile', renderProfilePage);
router.register('/club/:clubId', renderClubDetailPage);
router.register('/club/:clubId/add-pick', renderAddPickPage);
router.register('/club/:clubId/pick/:pickId', renderPickDetailPage);

// Start router
router.init();
