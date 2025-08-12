export default {
    name: 'SystemDashboard',
    data() {
        return {
            systemOverview: null,
            loading: false,
            refreshInterval: null,
            autoRefresh: true
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="dashboard icon"></i>
                        System Dashboard
                    </div>
                    <div class="description">
                        Complete system overview and health status
                    </div>
                </div>
                <div class="content">
                    <div class="ui buttons">
                        <button class="ui primary button" @click="loadSystemOverview" :class="{ loading: loading }">
                            <i class="refresh icon"></i>
                            Refresh
                        </button>
                        <button class="ui button" @click="toggleAutoRefresh" :class="{ green: autoRefresh }">
                            <i class="clock icon"></i>
                            Auto Refresh
                        </button>
                    </div>
                </div>
                <div class="content" v-if="systemOverview">
                    <!-- System Health -->
                    <div class="ui segment">
                        <h4 class="ui header">System Health</h4>
                        <div class="ui four statistics">
                            <div class="statistic">
                                <div class="value" :class="{ green: systemOverview.health.overall === 'healthy', red: systemOverview.health.overall === 'unhealthy' }">
                                    <i :class="systemOverview.health.overall === 'healthy' ? 'check circle icon' : 'exclamation triangle icon'"></i>
                                </div>
                                <div class="label">Overall Health</div>
                            </div>
                            <div class="statistic">
                                <div class="value">{{ systemOverview.health.uptime }}</div>
                                <div class="label">Uptime</div>
                            </div>
                            <div class="statistic">
                                <div class="value">{{ systemOverview.health.cpu_usage }}%</div>
                                <div class="label">CPU Usage</div>
                            </div>
                            <div class="statistic">
                                <div class="value">{{ systemOverview.health.memory_usage }}%</div>
                                <div class="label">Memory Usage</div>
                            </div>
                        </div>
                    </div>

                    <!-- WhatsApp Instances -->
                    <div class="ui segment">
                        <h4 class="ui header">WhatsApp Instances</h4>
                        <div class="ui four statistics">
                            <div class="statistic">
                                <div class="value">{{ systemOverview.instances.total }}</div>
                                <div class="label">Total Instances</div>
                            </div>
                            <div class="statistic">
                                <div class="value green">{{ systemOverview.instances.running }}</div>
                                <div class="label">Running</div>
                            </div>
                            <div class="statistic">
                                <div class="value red">{{ systemOverview.instances.stopped }}</div>
                                <div class="label">Stopped</div>
                            </div>
                            <div class="statistic">
                                <div class="value">{{ systemOverview.instances.connected }}</div>
                                <div class="label">Connected</div>
                            </div>
                        </div>
                    </div>

                    <!-- Message Statistics -->
                    <div class="ui segment">
                        <h4 class="ui header">Message Statistics (Last 24h)</h4>
                        <div class="ui four statistics">
                            <div class="statistic">
                                <div class="value">{{ systemOverview.messages.total }}</div>
                                <div class="label">Total Messages</div>
                            </div>
                            <div class="statistic">
                                <div class="value green">{{ systemOverview.messages.sent }}</div>
                                <div class="label">Sent</div>
                            </div>
                            <div class="statistic">
                                <div class="value blue">{{ systemOverview.messages.received }}</div>
                                <div class="label">Received</div>
                            </div>
                            <div class="statistic">
                                <div class="value red">{{ systemOverview.messages.failed }}</div>
                                <div class="label">Failed</div>
                            </div>
                        </div>
                    </div>

                    <!-- Queue Status -->
                    <div class="ui segment">
                        <h4 class="ui header">Queue Status</h4>
                        <div class="ui four statistics">
                            <div class="statistic">
                                <div class="value yellow">{{ systemOverview.queue.pending }}</div>
                                <div class="label">Pending Jobs</div>
                            </div>
                            <div class="statistic">
                                <div class="value blue">{{ systemOverview.queue.processing }}</div>
                                <div class="label">Processing</div>
                            </div>
                            <div class="statistic">
                                <div class="value green">{{ systemOverview.queue.completed }}</div>
                                <div class="label">Completed</div>
                            </div>
                            <div class="statistic">
                                <div class="value red">{{ systemOverview.queue.failed }}</div>
                                <div class="label">Failed</div>
                            </div>
                        </div>
                    </div>

                    <!-- Storage & Cache -->
                    <div class="ui segment">
                        <h4 class="ui header">Storage & Cache</h4>
                        <div class="ui four statistics">
                            <div class="statistic">
                                <div class="value">{{ formatBytes(systemOverview.storage.used) }}</div>
                                <div class="label">Storage Used</div>
                            </div>
                            <div class="statistic">
                                <div class="value">{{ systemOverview.cache.keys }}</div>
                                <div class="label">Cache Keys</div>
                            </div>
                            <div class="statistic">
                                <div class="value">{{ systemOverview.cache.hit_rate }}%</div>
                                <div class="label">Cache Hit Rate</div>
                            </div>
                            <div class="statistic">
                                <div class="value">{{ systemOverview.backups.count }}</div>
                                <div class="label">Backups</div>
                            </div>
                        </div>
                    </div>

                    <!-- Recent Activity -->
                    <div class="ui segment">
                        <h4 class="ui header">Recent Activity</h4>
                        <div class="ui relaxed divided list">
                            <div class="item" v-for="activity in systemOverview.recent_activity" :key="activity.id">
                                <div class="right floated content">
                                    <div class="ui label" :class="getActivityColor(activity.type)">
                                        {{ activity.type }}
                                    </div>
                                </div>
                                <div class="content">
                                    <div class="header">{{ activity.title }}</div>
                                    <div class="description">
                                        {{ activity.description }}<br>
                                        <small>{{ formatDate(activity.timestamp) }}</small>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- System Alerts -->
                    <div class="ui segment" v-if="systemOverview.alerts && systemOverview.alerts.length > 0">
                        <h4 class="ui header">System Alerts</h4>
                        <div class="ui messages">
                            <div class="ui message" v-for="alert in systemOverview.alerts" :key="alert.id"
                                 :class="{ warning: alert.level === 'warning', error: alert.level === 'error', info: alert.level === 'info' }">
                                <div class="header">{{ alert.title }}</div>
                                <p>{{ alert.message }}</p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `,
    methods: {
        async loadSystemOverview() {
            this.loading = true;
            try {
                const response = await window.http.get('/system/overview');
                
                // Validate response structure
                if (response.data && response.data.results) {
                    this.systemOverview = response.data.results;
                    
                    // Ensure all required properties exist with defaults
                    if (!this.systemOverview.health) {
                        this.systemOverview.health = {
                            overall: 'unknown',
                            uptime: '0m',
                            cpu_usage: 0,
                            memory_usage: 0
                        };
                    }
                    
                    if (!this.systemOverview.instances) {
                        this.systemOverview.instances = {
                            total: 0,
                            running: 0,
                            stopped: 0,
                            connected: 0
                        };
                    }
                    
                    if (!this.systemOverview.messages) {
                        this.systemOverview.messages = {
                            total: 0,
                            sent: 0,
                            received: 0,
                            failed: 0
                        };
                    }
                    
                    if (!this.systemOverview.queue) {
                        this.systemOverview.queue = {
                            pending: 0,
                            processing: 0,
                            completed: 0,
                            failed: 0
                        };
                    }
                    
                    if (!this.systemOverview.storage) {
                        this.systemOverview.storage = { used: 0 };
                    }
                    
                    if (!this.systemOverview.cache) {
                        this.systemOverview.cache = {
                            keys: 0,
                            hit_rate: 0
                        };
                    }
                    
                    if (!this.systemOverview.backups) {
                        this.systemOverview.backups = { count: 0 };
                    }
                    
                    if (!this.systemOverview.recent_activity) {
                        this.systemOverview.recent_activity = [];
                    }
                    
                    if (!this.systemOverview.alerts) {
                        this.systemOverview.alerts = [];
                    }
                    
                    if (!this.autoRefresh) {
                        showSuccessInfo('System overview loaded successfully');
                    }
                } else {
                    throw new Error('Invalid response format');
                }
            } catch (error) {
                console.error('System overview error:', error);
                showErrorInfo('Failed to load system overview: ' + (error.response?.data?.message || error.message));
                
                // Set default values on error
                this.systemOverview = {
                    health: {
                        overall: 'error',
                        uptime: 'Unknown',
                        cpu_usage: 0,
                        memory_usage: 0
                    },
                    instances: { total: 0, running: 0, stopped: 0, connected: 0 },
                    messages: { total: 0, sent: 0, received: 0, failed: 0 },
                    queue: { pending: 0, processing: 0, completed: 0, failed: 0 },
                    storage: { used: 0 },
                    cache: { keys: 0, hit_rate: 0 },
                    backups: { count: 0 },
                    recent_activity: [],
                    alerts: [{
                        id: 'connection_error',
                        level: 'error',
                        title: 'Connection Error',
                        message: 'Unable to connect to system overview API. Please check your connection.'
                    }]
                };
            } finally {
                this.loading = false;
            }
        },
        toggleAutoRefresh() {
            this.autoRefresh = !this.autoRefresh;
            if (this.autoRefresh) {
                this.refreshInterval = setInterval(() => {
                    this.loadSystemOverview();
                }, 30000); // Refresh every 30 seconds
                showSuccessInfo('Auto refresh enabled');
            } else {
                if (this.refreshInterval) {
                    clearInterval(this.refreshInterval);
                    this.refreshInterval = null;
                }
                showSuccessInfo('Auto refresh disabled');
            }
        },
        getActivityColor(type) {
            const colors = {
                'message': 'blue',
                'instance': 'green',
                'error': 'red',
                'backup': 'purple',
                'webhook': 'orange',
                'queue': 'yellow'
            };
            return colors[type] || 'grey';
        },
        formatBytes(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        },
        formatDate(dateString) {
            return new Date(dateString).toLocaleString();
        }
    },
    mounted() {
        this.loadSystemOverview();
        this.toggleAutoRefresh(); // Start auto refresh by default
    },
    beforeUnmount() {
        if (this.refreshInterval) {
            clearInterval(this.refreshInterval);
        }
    }
}