const WS_URL = `ws://${window.location.host}/ws`;

let ws = null;
let myClientId = null;
let myUsername = '';
let roomCode = '';
let isHost = false;
let gameState = null;
let canPlay = true;
let selectedPrefixCards = [];
let selectedSuffixCards = [];
let draggedCard = null;
let dragSource = null;

function connect() {
    ws = new WebSocket(WS_URL);

    ws.onopen = () => {
        console.log('Connected to server');
        showToast('Terhubung ke server', 'success');
    };

    ws.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        console.log('Received:', msg);
        handleMessage(msg);
    };

    ws.onclose = () => {
        console.log('Disconnected');
        showToast('Koneksi terputus', 'error');
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
}

function handleMessage(msg) {
    switch (msg.type) {
        case 'room_created':
            roomCode = msg.payload.roomCode;
            myClientId = msg.payload.playerId;
            isHost = true;
            showScreen('lobby');
            updateLobby();
            showToast('Room dibuat!', 'success');
            break;

        case 'connection_info':
            myClientId = msg.payload.playerId;
            console.log("Got playerId:", myClientId);
            break;

        case 'room_state':
            gameState = msg.payload;
            if (gameState.status === 'waiting') {
                showScreen('lobby');
                updateLobby();
            } else if (gameState.status === 'playing') {
                updateGame();
            }
            break;

        case 'game_state':
            gameState = msg.payload;
            if (gameState.status === 'playing') {
                updateGame();
                updateLeaderboard();
            }
            break;

        case 'play_result':
            handlePlayResult(msg.payload);
            break;

        case 'game_over':
            showToast(`PERMAINAN SELESAI! Pemenang: ${msg.payload.winnerName}`, 'success');
            showScreen('lobby');
            break;

        case 'error':
            showToast(msg.payload.message, 'error');
            break;

        case 'vote_request':
            handleVoteRequest(msg.payload);
            break;

        case 'vote_progress':
            handleVoteProgress(msg.payload);
            break;

        case 'vote_result':
            handleVoteResult(msg.payload);
            break;
    }
}

let voteTimer = null;
let currentVoteSeconds = 5;

function createRoom() {
    const username = document.getElementById('create-username').value.trim();
    if (!username) {
        showToast('Masukkan nama dulu', 'error');
        return;
    }

    myUsername = username;
    connect();

    ws.onopen = () => {
        console.log('WS connected, sending create_room');
        send({ type: 'create_room', payload: { username } });
    };

    ws.onerror = (error) => {
        console.error('WS error:', error);
        showToast('Koneksi gagal', 'error');
    };
}

function joinRoom() {
    const username = document.getElementById('join-username').value.trim();
    const code = document.getElementById('join-code').value.trim().toUpperCase();

    if (!username || !code) {
        showToast('Masukkan nama dan kode room', 'error');
        return;
    }

    myUsername = username;
    roomCode = code;
    connect();

    ws.onopen = () => {
        console.log('WS connected, sending join_room');
        send({ type: 'join_room', payload: { username, roomCode: code } });
    };

    ws.onerror = (error) => {
        console.error('WS error:', error);
        showToast('Koneksi gagal', 'error');
    };
}

function startGame() {
    send({ type: 'start_game', payload: {} });
}

function updatePreview() {
    const mainCard = gameState?.mainCard || '--';
    document.getElementById('main-card').textContent = mainCard;
    document.getElementById('main-card-display').textContent = mainCard;

    updateZoneDisplay('prefix-zone', selectedPrefixCards, 'prefix');
    updateZoneDisplay('suffix-zone', selectedSuffixCards, 'suffix');

    updateWordPreview();

    const btnSubmit = document.getElementById('btn-submit');
    const hasSelection = selectedPrefixCards.length > 0 || selectedSuffixCards.length > 0;
    btnSubmit.disabled = !hasSelection || !canPlay;
}

function updateZoneDisplay(zoneId, cards, zoneType) {
    const zone = document.getElementById(zoneId);
    zone.innerHTML = '';
    zone.dataset.zone = zoneType;

    if (cards.length === 0) {
        zone.innerHTML = `<span class="drop-label">${zoneType === 'prefix' ? 'DEPAN' : 'BELAKANG'}</span>`;
        zone.classList.remove('has-card');
    } else {
        cards.forEach((card, index) => {
            const cardEl = document.createElement('div');
            cardEl.className = 'selected-card';
            cardEl.textContent = card;
            cardEl.dataset.syllable = card;
            cardEl.dataset.index = index;
            cardEl.draggable = true;
            
            cardEl.addEventListener('dragstart', (e) => {
                e.stopPropagation();
                draggedCard = card;
                dragSource = { zone: zoneType, index: index };
                e.target.classList.add('dragging');
                e.dataTransfer.setData('text/plain', card);
                e.dataTransfer.effectAllowed = 'move';
            });

            cardEl.addEventListener('dragend', (e) => {
                e.target.classList.remove('dragging');
                draggedCard = null;
                dragSource = null;
            });

            cardEl.addEventListener('click', (e) => {
                e.stopPropagation();
                removeCardFromZone(zoneType, index);
            });

            zone.appendChild(cardEl);
        });
        zone.classList.add('has-card');
    }
}

function removeCardFromZone(zoneType, index) {
    if (zoneType === 'prefix') {
        selectedPrefixCards.splice(index, 1);
    } else {
        selectedSuffixCards.splice(index, 1);
    }
    updatePreview();
    renderHand();
}

function updateWordPreview() {
    const prefix = selectedPrefixCards.join('');
    const suffix = selectedSuffixCards.join('');
    const main = gameState?.mainCard || '';
    const word = prefix + main + suffix;

    document.getElementById('word-preview').textContent = word || '---';
    document.getElementById('preview-prefix').textContent = prefix;
    document.getElementById('main-card').textContent = main;
    document.getElementById('preview-suffix').textContent = suffix;
}

function submitPlay() {
    if (selectedPrefixCards.length === 0 && selectedSuffixCards.length === 0) {
        showToast('Pilih kartu dulu', 'error');
        return;
    }

    send({
        type: 'play_cards',
        payload: {
            prefixCards: selectedPrefixCards,
            suffixCards: selectedSuffixCards
        }
    });
}

function drawCard() {
    send({ type: 'draw_card', payload: {} });
}

function requestChangeMain() {
    send({ type: 'request_change_main', payload: {} });
}

function submitVote(approved) {
    send({ type: 'vote_response', payload: { approved } });
    hideVotePopup();
}

function handleVoteRequest(payload) {
    const popup = document.getElementById('vote-popup');
    document.getElementById('vote-initiator').textContent = payload.initiatorName;
    document.getElementById('vote-approve').textContent = '✓ 0';
    document.getElementById('vote-reject').textContent = '✗ 0';

    currentVoteSeconds = payload.secondsLeft;
    document.getElementById('vote-timer').textContent = currentVoteSeconds;

    popup.classList.remove('hidden');

    if (voteTimer) clearInterval(voteTimer);
    voteTimer = setInterval(() => {
        currentVoteSeconds--;
        document.getElementById('vote-timer').textContent = currentVoteSeconds;
        if (currentVoteSeconds <= 0) {
            clearInterval(voteTimer);
            hideVotePopup();
        }
    }, 1000);
}

function handleVoteProgress(payload) {
    document.getElementById('vote-approve').textContent = '✓ ' + payload.approved;
    document.getElementById('vote-reject').textContent = '✗ ' + payload.rejected;
}

function handleVoteResult(payload) {
    hideVotePopup();
    showToast(payload.message, payload.success ? 'success' : 'error');
}

function hideVotePopup() {
    document.getElementById('vote-popup').classList.add('hidden');
    if (voteTimer) {
        clearInterval(voteTimer);
        voteTimer = null;
    }
}

function clearSelection() {
    selectedPrefixCards = [];
    selectedSuffixCards = [];
    updatePreview();
    renderHand();
}

function send(msg) {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(msg));
    }
}

function showScreen(name) {
    document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
    document.getElementById(`screen-${name}`).classList.add('active');
}

function showToast(text, type = '') {
    const toast = document.getElementById('toast');
    toast.textContent = text;
    toast.className = 'toast ' + type;

    setTimeout(() => {
        toast.classList.add('hidden');
    }, 3000);
}

function updateLobby() {
    if (!gameState) return;

    document.getElementById('lobby-room-code').textContent = gameState.roomCode;
    document.getElementById('player-count').textContent = gameState.players.length;

    const list = document.getElementById('players-list');
    list.innerHTML = gameState.players.map(p => `
        <li>
            <span>${p.username}</span>
            ${p.isHost ? '<span class="host-badge">HOST</span>' : ''}
        </li>
    `).join('');

    const startBtn = document.getElementById('btn-start');
    startBtn.disabled = !isHost || gameState.players.length < 1;
}

function updateGame() {
    showScreen('game');
    selectedPrefixCards = [];
    selectedSuffixCards = [];
    canPlay = true;
    updatePreview();
    renderHand();
    updateLeaderboard();
}

function updateLeaderboard() {
    const list = document.getElementById('leaderboard-list');
    if (!gameState || !gameState.leaderboard) {
        list.innerHTML = '<span class="no-data">Belum ada</span>';
        return;
    }

    const medals = ['1st', '2nd', '3rd'];
    list.innerHTML = gameState.leaderboard.map((player, index) => {
        const rankClass = index < 3 ? `rank-${index + 1}` : '';
        return `
            <span class="leaderboard-entry ${rankClass}">${index < 3 ? medals[index] : ''} ${player.username}: ${player.score}</span>
        `;
    }).join(' &nbsp;|&nbsp; ');
}

function renderHand() {
    const hand = document.getElementById('player-hand');
    hand.innerHTML = '';

    if (!gameState || !gameState.players) return;

    const me = gameState.players.find(p => p.id === myClientId);
    if (!me || !me.cards) return;

    me.cards.forEach(syllable => {
        const isSelected = selectedPrefixCards.includes(syllable) || selectedSuffixCards.includes(syllable);
        if (isSelected) return;

        const cardEl = document.createElement('div');
        cardEl.className = 'hand-card';
        cardEl.textContent = syllable;
        cardEl.dataset.syllable = syllable;
        cardEl.draggable = true;

        cardEl.addEventListener('dragstart', (e) => {
            draggedCard = syllable;
            dragSource = { zone: 'hand', card: syllable };
            e.target.classList.add('dragging');
            e.dataTransfer.setData('text/plain', syllable);
            e.dataTransfer.effectAllowed = 'move';
        });

        cardEl.addEventListener('dragend', (e) => {
            e.target.classList.remove('dragging');
            draggedCard = null;
            dragSource = null;
        });

        hand.appendChild(cardEl);
    });
}

function setupDropZones() {
    const prefixZone = document.getElementById('prefix-zone');
    const suffixZone = document.getElementById('suffix-zone');

    [prefixZone, suffixZone].forEach(zone => {
        const zoneType = zone.id === 'prefix-zone' ? 'prefix' : 'suffix';

        zone.addEventListener('dragover', (e) => {
            e.preventDefault();
            e.dataTransfer.dropEffect = 'move';
            zone.classList.add('drag-over');
        });

        zone.addEventListener('dragleave', (e) => {
            zone.classList.remove('drag-over');
        });

        zone.addEventListener('drop', (e) => {
            e.preventDefault();
            zone.classList.remove('drag-over');

            const card = e.dataTransfer.getData('text/plain');
            if (!card) return;

            if (dragSource && dragSource.zone !== 'hand') {
                removeCardFromZone(dragSource.zone, dragSource.index);
            }

            if (zoneType === 'prefix') {
                if (!selectedPrefixCards.includes(card)) {
                    selectedPrefixCards.push(card);
                }
            } else {
                if (!selectedSuffixCards.includes(card)) {
                    selectedSuffixCards.push(card);
                }
            }

            updatePreview();
            renderHand();
        });
    });

    document.getElementById('player-hand').addEventListener('dragover', (e) => {
        if (dragSource && dragSource.zone !== 'hand') {
            e.preventDefault();
            e.dataTransfer.dropEffect = 'move';
        }
    });

    document.getElementById('player-hand').addEventListener('drop', (e) => {
        e.preventDefault();
        if (dragSource && dragSource.zone !== 'hand') {
            removeCardFromZone(dragSource.zone, dragSource.index);
        }
    });
}

function handlePlayResult(payload) {
    canPlay = false;
    const messageEl = document.getElementById('play-message');

    messageEl.classList.remove('hidden', 'success', 'error');

    if (payload.valid) {
        messageEl.classList.add('success');
        messageEl.textContent = `${payload.playerName} menjawab BENAR! Kata: ${payload.word}`;

        document.getElementById('main-card').textContent = payload.newMainCard;
        document.getElementById('main-card-display').textContent = payload.newMainCard;
    } else {
        messageEl.classList.add('error');
        messageEl.textContent = `SALAH! ${payload.message}`;
    }

    updateLeaderboard();

    setTimeout(() => {
        canPlay = true;
        selectedPrefixCards = [];
        selectedSuffixCards = [];
        messageEl.classList.add('hidden');
        updatePreview();
        renderHand();
    }, 3000);
}

function disconnect() {
    if (ws) ws.close();
    ws = null;
    roomCode = '';
    isHost = false;
    gameState = null;
    canPlay = true;
    showScreen('landing');
}

function showWords() {
    fetch('/words')
        .then(res => res.text())
        .then(text => {
            const words = text.split('\n').filter(w => w.trim());
            const list = document.getElementById('words-list');
            list.innerHTML = words.map(w => `<span>${w}</span>`).join('');
            document.getElementById('words-modal').classList.remove('hidden');
        })
        .catch(err => {
            showToast('Gagal memuat kata', 'error');
        });
}

function closeWords() {
    document.getElementById('words-modal').classList.add('hidden');
}

document.addEventListener('DOMContentLoaded', () => {
    setupDropZones();
});