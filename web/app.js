const WS_URL = `ws://${window.location.host}/ws`;

let ws = null;
let myClientId = null;
let myUsername = '';
let roomCode = '';
let isHost = false;
let gameState = null;
let canPlay = true;
let draggedCard = null;
let selectedPosition = ''; // 'prefix' or 'suffix'
let selectedCard = ''; // Kartu yang dipilih

// Connection
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

        
    }
}

// Actions
function createRoom() {
    const username = document.getElementById('create-username').value.trim();
    if (!username) {
        showToast('Masukkan nama dulu', 'error');
        return;
    }

    myUsername = username;
    connect();

    ws.onopen = () => {
        send({ type: 'create_room', payload: { username } });
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
        send({ type: 'join_room', payload: { username, roomCode: code } });
    };
}

function startGame() {
    send({ type: 'start_game', payload: {} });
}

// Drag and Drop Functions
function dragStart(e, card) {
    draggedCard = card;
    e.dataTransfer.setData('text/plain', card);
    e.target.classList.add('dragging');
}

function dragEnd(e) {
    e.target.classList.remove('dragging');
    draggedCard = null;
}

function allowDrop(e) {
    e.preventDefault();
    const zone = e.currentTarget;
    zone.classList.add('drag-over');
}

function dragLeave(e) {
    e.currentTarget.classList.remove('drag-over');
}

function dropPrefix(e) {
    e.preventDefault();
    e.currentTarget.classList.remove('drag-over');
    
    if (!draggedCard || !canPlay) return;
    
    selectedPosition = 'prefix';
    selectedCard = draggedCard;
    updatePreview();
}

function dropSuffix(e) {
    e.preventDefault();
    e.currentTarget.classList.remove('drag-over');
    
    if (!draggedCard || !canPlay) return;
    
    selectedPosition = 'suffix';
    selectedCard = draggedCard;
    updatePreview();
}

// Update preview combination
function updatePreview() {
    const prefixEl = document.getElementById('prefix-zone');
    const suffixEl = document.getElementById('suffix-zone');
    const mainCard = gameState?.mainCard || '--';
    const btnSubmit = document.getElementById('btn-submit');
    
    // Update main card display
    document.getElementById('main-card').textContent = mainCard;
    
    // Clear zones
    prefixEl.innerHTML = '<span class="drop-label">DEPAN</span>';
    prefixEl.classList.remove('has-card');
    prefixEl.querySelector('.dropped-card')?.remove();
    
    suffixEl.innerHTML = '<span class="drop-label">BELAKANG</span>';
    suffixEl.classList.remove('has-card');
    suffixEl.querySelector('.dropped-card')?.remove();
    
    // Show selected card
    if (selectedPosition === 'prefix' && selectedCard) {
        const cardEl = document.createElement('span');
        cardEl.className = 'dropped-card';
        cardEl.textContent = selectedCard;
        prefixEl.appendChild(cardEl);
        prefixEl.classList.add('has-card');
        prefixEl.innerHTML += '<span class="dropped-card">' + selectedCard + '</span>';
    } else if (selectedPosition === 'suffix' && selectedCard) {
        suffixEl.innerHTML = '<span class="drop-label">BELAKANG</span><span class="dropped-card">' + selectedCard + '</span>';
        suffixEl.classList.add('has-card');
    }
    
    // Enable submit if have selection
    btnSubmit.disabled = !selectedCard || !canPlay;
}

function submitPlay() {
    if (!selectedCard || !selectedPosition || !canPlay) {
        showToast('Pilih posisi dulu', 'error');
        return;
    }

    send({
        type: 'play_cards',
        payload: {
            cards: [selectedCard],
            position: selectedPosition
        }
    });
}

function drawCard() {
    send({ type: 'draw_card', payload: {} });
}

function changeMainCard() {
    send({ type: 'change_main_card', payload: {} });
}

function clearSelection() {
    selectedPosition = '';
    selectedCard = '';
    updatePreview();
}

// Helpers
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
    selectedPosition = '';
    selectedCard = '';
    canPlay = true;
    updatePreview();
    renderHand();
}

function renderHand() {
    const hand = document.getElementById('player-hand');
    hand.innerHTML = '';

    if (!gameState || !gameState.players) return;

    const me = gameState.players.find(p => p.id === myClientId);
    if (!me || !me.cards) return;

    me.cards.forEach(card => {
        const el = document.createElement('div');
        el.className = 'hand-card';
        if (selectedCard === card) {
            el.classList.add('selected');
        }
        el.textContent = card;
        el.draggable = true;
        el.ondragstart = (e) => dragStart(e, card);
        el.ondragend = dragEnd;
        hand.appendChild(el);
    });
}



function handlePlayResult(payload) {
    canPlay = false;
    const prefixZone = document.getElementById('prefix-zone');
    const suffixZone = document.getElementById('suffix-zone');
    const messageEl = document.getElementById('play-message');
    
    messageEl.classList.remove('hidden', 'success', 'error');
    
    if (payload.valid) {
        messageEl.classList.add('success');
        messageEl.textContent = `${payload.playerName} menjawab BENAR! Kata: ${payload.word}`;
        
        if (selectedPosition === 'prefix') {
            prefixZone.classList.add('correct');
        } else {
            suffixZone.classList.add('correct');
        }
        
        document.getElementById('main-card').textContent = payload.newMainCard;
    } else {
        messageEl.classList.add('error');
        messageEl.textContent = `SALAH! ${payload.message}`;
        
        if (selectedPosition === 'prefix') {
            prefixZone.classList.add('wrong');
        } else {
            suffixZone.classList.add('wrong');
        }
    }

    setTimeout(() => {
        canPlay = true;
        selectedPosition = '';
        selectedCard = '';
        prefixZone.classList.remove('correct', 'wrong');
        suffixZone.classList.remove('correct', 'wrong');
        messageEl.classList.add('hidden');
        
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

// Word List Functions
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