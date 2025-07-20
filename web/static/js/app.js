// Application state
let state = {
    isRunning: false,
    config: {},
    stats: {},
    chartInstance: null,
    updateInterval: null
};

// Initialize the application
document.addEventListener('DOMContentLoaded', function() {
    initTabs();
    loadConfig();
    loadStats();
    setupEventListeners();
    startStatsUpdate();
});

// Tab management
function initTabs() {
    const tabButtons = document.querySelectorAll('.tab-button');
    const tabContents = document.querySelectorAll('.tab-content');

    tabButtons.forEach(button => {
        button.addEventListener('click', () => {
            const targetTab = button.dataset.tab;
            
            // Remove active class from all tabs and buttons
            tabButtons.forEach(btn => btn.classList.remove('active'));
            tabContents.forEach(content => content.classList.remove('active'));
            
            // Add active class to clicked button and corresponding content
            button.classList.add('active');
            document.getElementById(targetTab).classList.add('active');
        });
    });
}

// Load configuration from server
async function loadConfig() {
    try {
        const response = await fetch('/api/config');
        if (response.ok) {
            const config = await response.json();
            state.config = config;
            populateConfigForm(config);
        }
    } catch (error) {
        console.error('Error loading config:', error);
        showAlert('error', 'Ошибка загрузки конфигурации');
    }
}

// Populate configuration form with data
function populateConfigForm(config) {
    const form = document.getElementById('config-form');
    if (!form) return;

    Object.keys(config).forEach(key => {
        const element = form.querySelector(`[name="${key}"]`);
        if (element) {
            if (element.type === 'checkbox') {
                element.checked = config[key];
            } else {
                element.value = config[key];
            }
        }
    });
}

// Save configuration
async function saveConfig() {
    const form = document.getElementById('config-form');
    if (!form) return;

    const formData = new FormData(form);
    const config = {};

    // Convert form data to config object
    for (let [key, value] of formData.entries()) {
        const element = form.querySelector(`[name="${key}"]`);
        if (element) {
            if (element.type === 'checkbox') {
                config[key] = element.checked;
            } else if (element.type === 'number') {
                config[key] = parseFloat(value) || 0;
            } else {
                config[key] = value;
            }
        }
    }

    try {
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(config)
        });

        if (response.ok) {
            state.config = config;
            showAlert('success', 'Конфигурация сохранена успешно');
        } else {
            showAlert('error', 'Ошибка сохранения конфигурации');
        }
    } catch (error) {
        console.error('Error saving config:', error);
        showAlert('error', 'Ошибка сохранения конфигурации');
    }
}

// Load statistics from server
async function loadStats() {
    try {
        const response = await fetch('/api/stats');
        if (response.ok) {
            const stats = await response.json();
            state.stats = stats;
            updateStatsDisplay(stats);
        }
    } catch (error) {
        console.error('Error loading stats:', error);
    }
}

// Update statistics display
function updateStatsDisplay(stats) {
    // Update stat cards
    updateStatCard('total-clients', stats.TotalClients || 0);
    updateStatCard('packets-sent', stats.TotalPacketsSent || 0);
    updateStatCard('packets-received', stats.TotalPacketsRcvd || 0);
    updateStatCard('total-sync', stats.TotalSyncRcvd || 0);
    updateStatCard('total-announce', stats.TotalAnnounceRcvd || 0);
    updateStatCard('total-delay-req', stats.TotalDelayReqSent || 0);
    updateStatCard('pfring-rx-packets', stats.PFRingRXPackets || 0);
    updateStatCard('pfring-tx-packets', stats.PFRingTXPackets || 0);
    updateStatCard('pfring-dropped', stats.PFRingRXDropped || 0);
    updateStatCard('hw-timestamps', stats.PFRingHWTimestamps || 0);

    // Update status
    const statusElement = document.getElementById('system-status');
    if (statusElement) {
        statusElement.innerHTML = getStatusIndicator(stats);
    }

    // Update charts if visible
    updateCharts(stats);
}

// Update individual stat card
function updateStatCard(id, value) {
    const element = document.getElementById(id);
    if (element) {
        element.textContent = formatNumber(value);
    }
}

// Format numbers for display
function formatNumber(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    } else if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

// Get status indicator HTML
function getStatusIndicator(stats) {
    if (state.isRunning) {
        return '<span class="status-indicator status-running">● Работает</span>';
    } else {
        return '<span class="status-indicator status-stopped">● Остановлен</span>';
    }
}

// Update charts
function updateCharts(stats) {
    const chartContainer = document.getElementById('performance-chart');
    if (chartContainer && typeof Chart !== 'undefined') {
        updatePerformanceChart(stats);
    }
}

// Update performance chart
function updatePerformanceChart(stats) {
    const ctx = document.getElementById('performance-chart');
    if (!ctx) return;

    if (state.chartInstance) {
        // Update existing chart
        const now = new Date();
        state.chartInstance.data.labels.push(now.toLocaleTimeString());
        state.chartInstance.data.datasets[0].data.push(stats.TotalPacketsSent || 0);
        state.chartInstance.data.datasets[1].data.push(stats.TotalPacketsRcvd || 0);
        
        // Keep only last 20 data points
        if (state.chartInstance.data.labels.length > 20) {
            state.chartInstance.data.labels.shift();
            state.chartInstance.data.datasets.forEach(dataset => dataset.data.shift());
        }
        
        state.chartInstance.update();
    } else {
        // Create new chart
        state.chartInstance = new Chart(ctx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Отправлено пакетов',
                    data: [],
                    borderColor: '#2563eb',
                    backgroundColor: 'rgba(37, 99, 235, 0.1)',
                    tension: 0.4
                }, {
                    label: 'Получено пакетов',
                    data: [],
                    borderColor: '#059669',
                    backgroundColor: 'rgba(5, 150, 105, 0.1)',
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                plugins: {
                    legend: {
                        labels: {
                            color: '#cbd5e1'
                        }
                    }
                },
                scales: {
                    x: {
                        ticks: {
                            color: '#94a3b8'
                        },
                        grid: {
                            color: '#475569'
                        }
                    },
                    y: {
                        ticks: {
                            color: '#94a3b8'
                        },
                        grid: {
                            color: '#475569'
                        }
                    }
                }
            }
        });
    }
}

// Start/Stop system
async function toggleSystem() {
    const button = document.getElementById('toggle-system');
    if (!button) return;

    button.disabled = true;
    button.innerHTML = '<span class="loading"></span> Обработка...';

    try {
        const endpoint = state.isRunning ? '/api/stop' : '/api/start';
        const response = await fetch(endpoint, { method: 'POST' });

        if (response.ok) {
            state.isRunning = !state.isRunning;
            updateSystemButton();
            showAlert('success', state.isRunning ? 'Система запущена' : 'Система остановлена');
        } else {
            showAlert('error', 'Ошибка управления системой');
        }
    } catch (error) {
        console.error('Error toggling system:', error);
        showAlert('error', 'Ошибка управления системой');
    } finally {
        button.disabled = false;
    }
}

// Update system control button
function updateSystemButton() {
    const button = document.getElementById('toggle-system');
    if (!button) return;

    if (state.isRunning) {
        button.className = 'btn btn-danger';
        button.innerHTML = '⏹ Остановить';
    } else {
        button.className = 'btn btn-success';
        button.innerHTML = '▶ Запустить';
    }
}

// Show alert message
function showAlert(type, message) {
    const alertContainer = document.getElementById('alert-container');
    if (!alertContainer) return;

    const alert = document.createElement('div');
    alert.className = `alert alert-${type}`;
    alert.textContent = message;

    alertContainer.appendChild(alert);

    // Auto-remove after 5 seconds
    setTimeout(() => {
        alert.remove();
    }, 5000);
}

// Setup event listeners
function setupEventListeners() {
    // Config form save button
    const saveConfigBtn = document.getElementById('save-config');
    if (saveConfigBtn) {
        saveConfigBtn.addEventListener('click', saveConfig);
    }

    // System toggle button
    const toggleSystemBtn = document.getElementById('toggle-system');
    if (toggleSystemBtn) {
        toggleSystemBtn.addEventListener('click', toggleSystem);
    }

    // Export config button
    const exportConfigBtn = document.getElementById('export-config');
    if (exportConfigBtn) {
        exportConfigBtn.addEventListener('click', exportConfig);
    }

    // Import config button
    const importConfigBtn = document.getElementById('import-config');
    if (importConfigBtn) {
        importConfigBtn.addEventListener('click', () => {
            document.getElementById('import-file').click();
        });
    }

    // File input for import
    const importFile = document.getElementById('import-file');
    if (importFile) {
        importFile.addEventListener('change', importConfig);
    }

    // Clear stats button
    const clearStatsBtn = document.getElementById('clear-stats');
    if (clearStatsBtn) {
        clearStatsBtn.addEventListener('click', clearStats);
    }
}

// Export configuration
function exportConfig() {
    const config = state.config;
    const dataStr = JSON.stringify(config, null, 2);
    const dataUri = 'data:application/json;charset=utf-8,'+ encodeURIComponent(dataStr);
    
    const exportFileDefaultName = 'clientgen_config.json';
    
    const linkElement = document.createElement('a');
    linkElement.setAttribute('href', dataUri);
    linkElement.setAttribute('download', exportFileDefaultName);
    linkElement.click();
}

// Import configuration
function importConfig(event) {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = function(e) {
        try {
            const config = JSON.parse(e.target.result);
            state.config = config;
            populateConfigForm(config);
            showAlert('success', 'Конфигурация импортирована успешно');
        } catch (error) {
            showAlert('error', 'Ошибка импорта конфигурации');
        }
    };
    reader.readAsText(file);
}

// Clear statistics
async function clearStats() {
    if (!confirm('Вы уверены, что хотите очистить статистику?')) {
        return;
    }

    try {
        const response = await fetch('/api/stats/clear', { method: 'POST' });
        if (response.ok) {
            showAlert('success', 'Статистика очищена');
            loadStats();
        } else {
            showAlert('error', 'Ошибка очистки статистики');
        }
    } catch (error) {
        console.error('Error clearing stats:', error);
        showAlert('error', 'Ошибка очистки статистики');
    }
}

// Start automatic stats updates
function startStatsUpdate() {
    if (state.updateInterval) {
        clearInterval(state.updateInterval);
    }
    
    state.updateInterval = setInterval(loadStats, 1000); // Update every second
}

// Stop automatic stats updates
function stopStatsUpdate() {
    if (state.updateInterval) {
        clearInterval(state.updateInterval);
        state.updateInterval = null;
    }
}