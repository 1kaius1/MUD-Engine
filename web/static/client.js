/**
 * MUD Web Client - JavaScript
 * Handles WebSocket connection, command input, and terminal display
 */

// Configuration
const CONFIG = {
    reconnectDelay: 3000,              // Delay before reconnection attempt (ms)
    maxReconnectAttempts: 5,           // Max automatic reconnection attempts
    historySize: 50,                   // Number of commands to keep in history
    defaultFontSize: 14,               // Default terminal font size
    defaultLineHeight: 1.0             // Default terminal line height
};

// User preferences (loaded from localStorage)
const preferences = {
    zoom: 100,                         // Zoom percentage (75, 100, 125, 150, 200)
    fontSize: CONFIG.defaultFontSize,  // Calculated font size
    lineHeight: CONFIG.defaultLineHeight
};

// Connection state
const connection = {
    ws: null,
    url: null,
    hostname: null,
    port: null,
    username: null,
    connectedAt: null,
    bytesSent: 0,
    bytesReceived: 0,
    serverInfo: null,
    isConnected: false,
    reconnectAttempts: 0
};

// Client state
const state = {
    commandHistory: [],
    historyIndex: -1,
    isPasswordMode: false
};

// DOM elements
let terminal, input, status;

/**
 * Initialize the client when DOM is ready
 */
document.addEventListener('DOMContentLoaded', function() {
    // Get DOM elements
    terminal = document.getElementById('terminal');
    input = document.getElementById('input');
    status = document.getElementById('status');
    
    // Load user preferences
    loadPreferences();
    
    // Apply preferences
    applyPreferences();
    
    // Set up event listeners
    setupEventListeners();
    
    // Display welcome message
    displayWelcome();
    
    // Focus input field
    input.focus();
});

/**
 * Display welcome message
 */
function displayWelcome() {
    appendToTerminal('╔════════════════════════════════════════╗\n');
    appendToTerminal('║     MUD Web Client v0.1                ║\n');
    appendToTerminal('║                                        ║\n');
    appendToTerminal('║     Waiting for server connection     ║\n');
    appendToTerminal('╚════════════════════════════════════════╝\n\n');
    appendToTerminal('Use /connect <host> <port> to connect to a MUD server.\n');
    appendToTerminal('Type /help for a list of available commands.\n\n');
    updateStatus('disconnected', 'Not connected');
}

/**
 * Set up event listeners for user interaction
 */
function setupEventListeners() {
    // Handle input submission and history navigation
    input.addEventListener('keydown', handleKeyDown);
    
    // Keep input focused when clicking anywhere
    document.addEventListener('click', function() {
        input.focus();
    });
}

/**
 * Handle keyboard input
 */
function handleKeyDown(e) {
    switch(e.key) {
        case 'Enter':
            handleSubmit();
            break;
            
        case 'ArrowUp':
            e.preventDefault();
            navigateHistory('up');
            break;
            
        case 'ArrowDown':
            e.preventDefault();
            navigateHistory('down');
            break;
            
        case 'Tab':
            // Future: tab completion
            e.preventDefault();
            break;
    }
}

/**
 * Handle command submission
 */
function handleSubmit() {
    const message = input.value;
    
    // Don't send empty messages
    if (message.trim() === '') {
        return;
    }
    
    // Check if this is a client command
    if (message.startsWith('/')) {
        handleClientCommand(message);
    } else {
        // Regular game command - send to server
        handleGameCommand(message);
    }
    
    // Clear input
    input.value = '';
    state.historyIndex = -1;
}

/**
 * Handle client commands (those starting with /)
 */
function handleClientCommand(command) {
    const parts = command.slice(1).split(/\s+/);
    const cmd = parts[0].toLowerCase();
    const args = parts.slice(1);
    
    // Echo command to terminal
    appendToTerminal(command + '\n');
    
    switch(cmd) {
        case 'connect':
            cmdConnect(args);
            break;
            
        case 'quit':
        case 'disconnect':
            cmdQuit();
            break;
            
        case 'help':
            cmdHelp();
            break;
            
        case 'status':
            cmdStatus();
            break;
            
        case 'info':
            cmdInfo();
            break;
            
        case 'clear':
            clearTerminal();
            break;
            
        case 'zoom':
            cmdZoom(args);
            break;
            
        case 'settings':
            cmdSettings(args);
            break;
            
        default:
            appendToTerminal(`Unknown command: /${cmd}\n`);
            appendToTerminal('Type /help for a list of available commands.\n\n');
    }
}

/**
 * Handle game commands (sent to server)
 */
function handleGameCommand(message) {
    // Echo command to terminal (unless in password mode)
    if (!state.isPasswordMode) {
        // Only echo if we're at a prompt
        if (terminal.textContent.endsWith('> ') || 
            terminal.textContent.endsWith(': ')) {
            appendToTerminal(message + '\n');
        }
    }
    
    // Send message to server
    sendMessage(message);
    
    // Add to command history (unless password)
    if (!state.isPasswordMode && message.trim() !== '') {
        state.commandHistory.unshift(message);
        if (state.commandHistory.length > CONFIG.historySize) {
            state.commandHistory.pop();
        }
    }
}

/**
 * /connect command - Connect to a MUD server
 */
function cmdConnect(args) {
    if (connection.isConnected) {
        appendToTerminal('Already connected. Use /quit to disconnect first.\n\n');
        return;
    }
    
    if (args.length < 2) {
        appendToTerminal('Usage: /connect <hostname> <port>\n');
        appendToTerminal('Example: /connect localhost 8080\n\n');
        return;
    }
    
    const hostname = args[0];
    const port = parseInt(args[1]);
    
    if (isNaN(port) || port < 1 || port > 65535) {
        appendToTerminal('Error: Invalid port number. Must be between 1 and 65535.\n\n');
        return;
    }
    
    connect(hostname, port);
}

/**
 * /quit command - Disconnect from server
 */
function cmdQuit() {
    if (!connection.isConnected) {
        appendToTerminal('Not connected to any server.\n\n');
        return;
    }
    
    appendToTerminal('Disconnecting from server...\n');
    disconnect();
}

/**
 * /help command - Display help information
 */
function cmdHelp() {
    appendToTerminal('═══════════════════════════════════════════════════════════════\n');
    appendToTerminal('                    CLIENT COMMANDS HELP                       \n');
    appendToTerminal('═══════════════════════════════════════════════════════════════\n\n');
    
    appendToTerminal('/connect <host> <port>\n');
    appendToTerminal('  Connect to a MUD server.\n');
    appendToTerminal('  Example: /connect localhost 8080\n\n');
    
    appendToTerminal('/quit (or /disconnect)\n');
    appendToTerminal('  Gracefully disconnect from the current server.\n\n');
    
    appendToTerminal('/status\n');
    appendToTerminal('  Display connection status and statistics.\n\n');
    
    appendToTerminal('/info\n');
    appendToTerminal('  Display information about the connected server.\n\n');
    
    appendToTerminal('/help\n');
    appendToTerminal('  Display this help message.\n\n');
    
    appendToTerminal('/clear\n');
    appendToTerminal('  Clear the terminal screen.\n\n');
    
    appendToTerminal('/zoom <percentage>\n');
    appendToTerminal('  Adjust terminal text size.\n');
    appendToTerminal('  Values: 75, 100, 125, 150, 200\n');
    appendToTerminal('  Example: /zoom 150\n\n');
    
    appendToTerminal('/settings\n');
    appendToTerminal('  Display current settings and preferences.\n\n');
    
    appendToTerminal('═══════════════════════════════════════════════════════════════\n');
    appendToTerminal('NOTE: Commands without / are sent to the game server.\n');
    appendToTerminal('═══════════════════════════════════════════════════════════════\n\n');
}

/**
 * /status command - Display connection status
 */
function cmdStatus() {
    appendToTerminal('═══════════════════════════════════════════════════════════════\n');
    appendToTerminal('                    CONNECTION STATUS                          \n');
    appendToTerminal('═══════════════════════════════════════════════════════════════\n\n');
    
    if (!connection.isConnected) {
        appendToTerminal('Status:      Not connected\n\n');
        return;
    }
    
    appendToTerminal(`Status:      Connected\n`);
    appendToTerminal(`Server:      ${connection.hostname}:${connection.port}\n`);
    
    if (connection.username) {
        appendToTerminal(`Username:    ${connection.username}\n`);
    }
    
    if (connection.connectedAt) {
        const duration = Math.floor((Date.now() - connection.connectedAt) / 1000);
        const hours = Math.floor(duration / 3600);
        const minutes = Math.floor((duration % 3600) / 60);
        const seconds = duration % 60;
        appendToTerminal(`Connected:   ${hours}h ${minutes}m ${seconds}s\n`);
    }
    
    appendToTerminal(`Bytes Sent:  ${connection.bytesSent.toLocaleString()}\n`);
    appendToTerminal(`Bytes Recv:  ${connection.bytesReceived.toLocaleString()}\n`);
    
    appendToTerminal('\n');
}

/**
 * /info command - Display server information
 */
function cmdInfo() {
    appendToTerminal('═══════════════════════════════════════════════════════════════\n');
    appendToTerminal('                    SERVER INFORMATION                         \n');
    appendToTerminal('═══════════════════════════════════════════════════════════════\n\n');
    
    if (!connection.isConnected) {
        appendToTerminal('Not connected to any server.\n\n');
        return;
    }
    
    if (!connection.serverInfo) {
        appendToTerminal('No server information available.\n');
        appendToTerminal('(This feature requires server support)\n\n');
        return;
    }
    
    // Display server info (to be implemented when server sends it)
    appendToTerminal(`Server Name: ${connection.serverInfo.name || 'Unknown'}\n`);
    appendToTerminal(`Version:     ${connection.serverInfo.version || 'Unknown'}\n`);
    appendToTerminal(`Theme:       ${connection.serverInfo.theme || 'Unknown'}\n`);
    appendToTerminal(`Description: ${connection.serverInfo.description || 'None'}\n\n`);
}

/**
 * /zoom command - Adjust terminal font size
 */
function cmdZoom(args) {
    if (args.length === 0) {
        appendToTerminal(`Current zoom: ${preferences.zoom}%\n`);
        appendToTerminal('Available zoom levels: 75, 100, 125, 150, 200\n');
        appendToTerminal('Usage: /zoom <percentage>\n');
        appendToTerminal('Example: /zoom 150\n\n');
        return;
    }
    
    const zoom = parseInt(args[0]);
    const validZooms = [75, 100, 125, 150, 200];
    
    if (isNaN(zoom) || !validZooms.includes(zoom)) {
        appendToTerminal('Error: Invalid zoom level.\n');
        appendToTerminal('Valid options: 75, 100, 125, 150, 200\n\n');
        return;
    }
    
    preferences.zoom = zoom;
    preferences.fontSize = CONFIG.defaultFontSize * (zoom / 100);
    
    applyPreferences();
    savePreferences();
    
    appendToTerminal(`Zoom set to ${zoom}%\n`);
    appendToTerminal(`Font size: ${preferences.fontSize}px\n\n`);
}

/**
 * /settings command - Display current settings
 */
function cmdSettings(args) {
    appendToTerminal('═══════════════════════════════════════════════════════════════\n');
    appendToTerminal('                    CLIENT SETTINGS                            \n');
    appendToTerminal('═══════════════════════════════════════════════════════════════\n\n');
    
    appendToTerminal('Display Settings:\n');
    appendToTerminal(`  Zoom Level:      ${preferences.zoom}%\n`);
    appendToTerminal(`  Font Size:       ${preferences.fontSize}px\n`);
    appendToTerminal(`  Line Height:     ${preferences.lineHeight}\n\n`);
    
    appendToTerminal('Connection Settings:\n');
    appendToTerminal(`  Reconnect Delay: ${CONFIG.reconnectDelay}ms\n`);
    appendToTerminal(`  Max Reconnects:  ${CONFIG.maxReconnectAttempts}\n\n`);
    
    appendToTerminal('History:\n');
    appendToTerminal(`  Command History: ${state.commandHistory.length}/${CONFIG.historySize}\n\n`);
    
    appendToTerminal('Tip: Use /zoom <percentage> to adjust text size.\n');
    appendToTerminal('Valid zoom levels: 75, 100, 125, 150, 200\n\n');
}

/**
 * Navigate through command history
 */
function navigateHistory(direction) {
    // Don't navigate history in password mode
    if (state.isPasswordMode) {
        return;
    }
    
    if (state.commandHistory.length === 0) {
        return;
    }
    
    if (direction === 'up') {
        if (state.historyIndex < state.commandHistory.length - 1) {
            state.historyIndex++;
            input.value = state.commandHistory[state.historyIndex];
        }
    } else if (direction === 'down') {
        if (state.historyIndex > 0) {
            state.historyIndex--;
            input.value = state.commandHistory[state.historyIndex];
        } else {
            state.historyIndex = -1;
            input.value = '';
        }
    }
}

/**
 * Connect to WebSocket server
 */
function connect(hostname, port) {
    updateStatus('connecting', `Connecting to ${hostname}:${port}...`);
    appendToTerminal(`Connecting to ${hostname}:${port}...\n`);
    
    connection.hostname = hostname;
    connection.port = port;
    connection.url = `ws://${hostname}:${port}/ws`;
    
    try {
        connection.ws = new WebSocket(connection.url);
        
        connection.ws.onopen = handleOpen;
        connection.ws.onmessage = handleMessage;
        connection.ws.onerror = handleError;
        connection.ws.onclose = handleClose;
    } catch (error) {
        console.error('WebSocket connection error:', error);
        appendToTerminal(`Connection failed: ${error.message}\n\n`);
        updateStatus('disconnected', 'Connection failed');
        resetConnection();
    }
}

/**
 * Disconnect from server
 */
function disconnect() {
    if (connection.ws) {
        connection.ws.close();
    }
    resetConnection();
}

/**
 * Reset connection state
 */
function resetConnection() {
    connection.isConnected = false;
    connection.ws = null;
    connection.connectedAt = null;
    connection.username = null;
    connection.serverInfo = null;
    connection.reconnectAttempts = 0;
    
    if (state.isPasswordMode) {
        state.isPasswordMode = false;
        input.type = 'text';
    }
}

/**
 * Handle WebSocket connection opened
 */
function handleOpen() {
    connection.isConnected = true;
    connection.connectedAt = Date.now();
    connection.bytesSent = 0;
    connection.bytesReceived = 0;
    
    updateStatus('connected', `Connected to ${connection.hostname}:${connection.port}`);
    appendToTerminal('Connected!\n\n');
    connection.reconnectAttempts = 0;
}

/**
 * Handle incoming WebSocket messages
 */
function handleMessage(event) {
    const message = event.data;
    connection.bytesReceived += message.length;
    
    // Check for username prompt to capture username
    if (message.includes('Login:') || message.includes('login:')) {
        // Next input will be username (we'll capture it on send)
    }
    
    // Check if we're entering password mode
    if (message.includes('Password:') && message.includes('\x1b[8m')) {
        state.isPasswordMode = true;
        input.type = 'password';
        const cleanMessage = message.replace(/\x1b\[[0-9;]*m/g, '');
        appendToTerminal(cleanMessage);
        return;
    }
    
    // Check if we're leaving password mode
    if (state.isPasswordMode && message.includes('\x1b[28m')) {
        state.isPasswordMode = false;
        input.type = 'text';
    }
    
    // Strip ANSI codes for now (we'll add color support later)
    const cleanMessage = stripAnsiCodes(message);
    appendToTerminal(cleanMessage);
}

/**
 * Handle WebSocket errors
 */
function handleError(error) {
    console.error('WebSocket error:', error);
    appendToTerminal('\n[Connection error occurred]\n\n');
}

/**
 * Handle WebSocket connection closed
 */
function handleClose(event) {
    const wasConnected = connection.isConnected;
    resetConnection();
    
    updateStatus('disconnected', 'Disconnected');
    
    if (wasConnected) {
        appendToTerminal('\n[Disconnected from server]\n\n');
    }
}

/**
 * Send message to server
 */
function sendMessage(message) {
    if (connection.ws && connection.ws.readyState === WebSocket.OPEN) {
        connection.ws.send(message);
        connection.bytesSent += message.length;
        
        // Capture username if this is the login prompt
        if (!connection.username && !state.isPasswordMode) {
            connection.username = message;
        }
    } else {
        appendToTerminal('[Error: Not connected to server]\n');
        appendToTerminal('Use /connect <host> <port> to connect.\n\n');
    }
}

/**
 * Append text to terminal
 */
function appendToTerminal(text) {
    terminal.textContent += text;
    
    // Auto-scroll to bottom
    terminal.scrollTop = terminal.scrollHeight;
}

/**
 * Update status bar
 */
function updateStatus(state, message) {
    status.textContent = message;
    status.className = state;
}

/**
 * Strip ANSI escape codes from text
 */
function stripAnsiCodes(text) {
    return text.replace(/\x1b\[[0-9;]*m/g, '');
}

/**
 * Clear terminal
 */
function clearTerminal() {
    terminal.textContent = '';
    appendToTerminal('Terminal cleared.\n\n');
}

/**
 * Handle visibility change
 */
document.addEventListener('visibilitychange', function() {
    if (!document.hidden) {
        input.focus();
    }
});

/**
 * Load user preferences from localStorage
 */
function loadPreferences() {
    try {
        const saved = localStorage.getItem('mudClientPreferences');
        if (saved) {
            const prefs = JSON.parse(saved);
            preferences.zoom = prefs.zoom || 100;
            preferences.fontSize = CONFIG.defaultFontSize * (preferences.zoom / 100);
            preferences.lineHeight = prefs.lineHeight || CONFIG.defaultLineHeight;
        }
    } catch (error) {
        console.error('Error loading preferences:', error);
        // Use defaults if loading fails
    }
}

/**
 * Save user preferences to localStorage
 */
function savePreferences() {
    try {
        const prefs = {
            zoom: preferences.zoom,
            lineHeight: preferences.lineHeight
        };
        localStorage.setItem('mudClientPreferences', JSON.stringify(prefs));
    } catch (error) {
        console.error('Error saving preferences:', error);
    }
}

/**
 * Apply current preferences to the terminal
 */
function applyPreferences() {
    const fontSize = preferences.fontSize + 'px';
    const lineHeight = preferences.lineHeight.toString();
    
    // Set CSS custom properties
    document.documentElement.style.setProperty('--terminal-font-size', fontSize);
    document.documentElement.style.setProperty('--terminal-line-height', lineHeight);
    
    // Also apply directly to terminal element (ensures immediate update)
    if (terminal) {
        terminal.style.fontSize = fontSize;
        terminal.style.lineHeight = lineHeight;
    }
    
    // Debug output
    console.log('Applied preferences:', {
        fontSize: fontSize,
        lineHeight: lineHeight,
        zoom: preferences.zoom + '%'
    });
}