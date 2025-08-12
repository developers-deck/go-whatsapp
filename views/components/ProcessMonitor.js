export default {
    name: 'ProcessMonitor',
    data() {
        return {
            processes: [],
            systemStats: null,
            loading: false,
            autoRefresh: false,
            refreshInterval: null
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="heartbeat icon"></i>
                        Process Monitor
                    </div>
                    <div class="description">
                        Monitor system processes and health
                    </div>
                </div>
                <div class="content">
                    <div class="ui buttons">
                        <button class="ui primary button" @click="loadProcesses" :class="{ loading: loading }">
                            <i class="refresh icon"></i>
                            Refresh
                        </button>
                        <button class="ui button" @click="toggleAutoRefresh" :class="{ green: autoRefresh }">
                            <i class="clock icon"></i>
                            Auto Refresh
                        </button>
                    </div>
                </div>
                <div class="content" v-if="systemStats">
                    <div class="ui statistics">
                        <div class="statistic">
                            <div class="value">{{ systemStats.cpu_usage }}%</div>
                            <div class="label">CPU Usage</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ systemStats.memory_usage }}%</div>
                            <div class="label">Memory Usage</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ systemStats.active_processes }}</div>
                            <div class="label">Active Processes</div>
                        </div>
                    </div>
                </div>
                <div class="content" v-if="processes.length > 0">
                    <table class="ui celled table">
                        <thead>
                            <tr>
                                <th>Process ID</th>
                                <th>Name</th>
                                <th>Status</th>
                                <th>CPU %</th>
                                <th>Memory</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="process in processes" :key="process.pid">
                                <td>{{ process.pid }}</td>
                                <td>{{ process.name }}</td>
                                <td>
                                    <span :class="{ 
                                        'ui green label': process.status === 'running',
                                        'ui red label': process.status === 'stopped',
                                        'ui yellow label': process.status === 'starting'
                                    }">
                                        {{ process.status }}
                                    </span>
                                </td>
                                <td>{{ process.cpu_percent }}%</td>
                                <td>{{ formatBytes(process.memory_usage) }}</td>
                                <td>
                                    <div class="ui buttons">
                                        <button class="ui small button" @click="restartProcess(process.pid)">
                                            <i class="redo icon"></i>
                                            Restart
                                        </button>
                                        <button class="ui small red button" @click="killProcess(process.pid)">
                                            <i class="stop icon"></i>
                                            Kill
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    `,
    methods: {
        async loadProcesses() {
            this.loading = true;
            try {
                const response = await window.http.get('/monitor/stats');
                this.processes = response.data.results || [];
                
                const statsResponse = await window.http.get('/monitor/stats');
                this.systemStats = statsResponse.data.results;
                
                if (!this.autoRefresh) {
                    showSuccessInfo('Process data loaded successfully');
                }
            } catch (error) {
                showErrorInfo('Failed to load process data: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        async restartProcess(pid) {
            try {
                await window.http.post(`/monitor/restart/${pid}`);
                showSuccessInfo('Process restarted successfully');
                this.loadProcesses();
            } catch (error) {
                showErrorInfo('Failed to restart process: ' + error.message);
            }
        },
        async killProcess(pid) {
            if (confirm('Are you sure you want to kill this process?')) {
                try {
                    await window.http.post(`/monitor/kill/${pid}`);
                    showSuccessInfo('Process killed successfully');
                    this.loadProcesses();
                } catch (error) {
                    showErrorInfo('Failed to kill process: ' + error.message);
                }
            }
        },
        toggleAutoRefresh() {
            this.autoRefresh = !this.autoRefresh;
            if (this.autoRefresh) {
                this.refreshInterval = setInterval(() => {
                    this.loadProcesses();
                }, 5000);
                showSuccessInfo('Auto refresh enabled');
            } else {
                if (this.refreshInterval) {
                    clearInterval(this.refreshInterval);
                    this.refreshInterval = null;
                }
                showSuccessInfo('Auto refresh disabled');
            }
        },
        formatBytes(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }
    },
    mounted() {
        this.loadProcesses();
    },
    beforeUnmount() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
    }
}