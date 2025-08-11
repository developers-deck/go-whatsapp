export default {
    name: 'CacheManager',
    data() {
        return {
            cacheStats: null,
            keys: [],
            loading: false,
            newKey: {
                key: '',
                value: '',
                ttl: 3600
            },
            showAddModal: false
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="database icon"></i>
                        Cache Manager
                    </div>
                    <div class="description">
                        Manage Redis cache and data
                    </div>
                </div>
                <div class="content">
                    <div class="ui buttons">
                        <button class="ui primary button" @click="loadCacheData" :class="{ loading: loading }">
                            <i class="refresh icon"></i>
                            Refresh
                        </button>
                        <button class="ui green button" @click="showAddModal = true">
                            <i class="plus icon"></i>
                            Add Key
                        </button>
                        <button class="ui red button" @click="flushCache">
                            <i class="trash icon"></i>
                            Flush All
                        </button>
                    </div>
                </div>
                <div class="content" v-if="cacheStats">
                    <div class="ui statistics">
                        <div class="statistic">
                            <div class="value">{{ cacheStats.total_keys }}</div>
                            <div class="label">Total Keys</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ formatBytes(cacheStats.memory_usage) }}</div>
                            <div class="label">Memory Usage</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ cacheStats.hit_rate }}%</div>
                            <div class="label">Hit Rate</div>
                        </div>
                    </div>
                </div>
                <div class="content" v-if="keys.length > 0">
                    <table class="ui celled table">
                        <thead>
                            <tr>
                                <th>Key</th>
                                <th>Type</th>
                                <th>TTL</th>
                                <th>Size</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="key in keys" :key="key.name">
                                <td>{{ key.name }}</td>
                                <td>{{ key.type }}</td>
                                <td>{{ key.ttl > 0 ? key.ttl + 's' : 'No expiry' }}</td>
                                <td>{{ formatBytes(key.size) }}</td>
                                <td>
                                    <div class="ui buttons">
                                        <button class="ui small button" @click="viewKey(key.name)">
                                            <i class="eye icon"></i>
                                            View
                                        </button>
                                        <button class="ui small red button" @click="deleteKey(key.name)">
                                            <i class="trash icon"></i>
                                            Delete
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
            
            <!-- Add Key Modal -->
            <div class="ui modal" :class="{ active: showAddModal }" id="addKeyModal">
                <div class="header">Add Cache Key</div>
                <div class="content">
                    <div class="ui form">
                        <div class="field">
                            <label>Key</label>
                            <input type="text" v-model="newKey.key" placeholder="Enter key name">
                        </div>
                        <div class="field">
                            <label>Value</label>
                            <textarea v-model="newKey.value" placeholder="Enter value" rows="3"></textarea>
                        </div>
                        <div class="field">
                            <label>TTL (seconds)</label>
                            <input type="number" v-model="newKey.ttl" placeholder="3600">
                        </div>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui cancel button" @click="closeAddModal">Cancel</div>
                    <div class="ui primary button" @click="addKey">Add Key</div>
                </div>
            </div>
            
            <!-- View Key Modal -->
            <div class="ui modal" :class="{ active: showViewModal }" id="viewKeyModal">
                <div class="header">View Key: {{ viewingKey.name }}</div>
                <div class="content">
                    <div class="ui segment">
                        <h4>Key Information</h4>
                        <p><strong>Type:</strong> {{ viewingKey.type }}</p>
                        <p><strong>TTL:</strong> {{ viewingKey.ttl > 0 ? viewingKey.ttl + 's' : 'No expiry' }}</p>
                        <p><strong>Size:</strong> {{ formatBytes(viewingKey.size) }}</p>
                    </div>
                    <div class="ui segment">
                        <h4>Value</h4>
                        <div class="ui code">{{ viewingKey.value }}</div>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui button" @click="showViewModal = false">Close</div>
                </div>
            </div>
        </div>
    `,
    data() {
        return {
            ...this.$options.data(),
            showViewModal: false,
            viewingKey: {}
        }
    },
    methods: {
        async loadCacheData() {
            this.loading = true;
            try {
                const statsResponse = await window.http.get('/cache/stats');
                this.cacheStats = statsResponse.data.results;
                
                const keysResponse = await window.http.get('/cache/keys');
                this.keys = keysResponse.data.results || [];
                
                showSuccessInfo('Cache data loaded successfully');
            } catch (error) {
                showErrorInfo('Failed to load cache data: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        async addKey() {
            try {
                await window.http.post('/cache/set', this.newKey);
                showSuccessInfo('Key added successfully');
                this.closeAddModal();
                this.loadCacheData();
            } catch (error) {
                showErrorInfo('Failed to add key: ' + error.message);
            }
        },
        async deleteKey(key) {
            if (confirm(\`Are you sure you want to delete key "\${key}"?\`)) {
                try {
                    await window.http.delete(\`/cache/keys/\${encodeURIComponent(key)}\`);
                    showSuccessInfo('Key deleted successfully');
                    this.loadCacheData();
                } catch (error) {
                    showErrorInfo('Failed to delete key: ' + error.message);
                }
            }
        },
        async viewKey(key) {
            try {
                const response = await window.http.get(\`/cache/keys/\${encodeURIComponent(key)}\`);
                this.viewingKey = response.data.results;
                this.showViewModal = true;
            } catch (error) {
                showErrorInfo('Failed to load key data: ' + error.message);
            }
        },
        async flushCache() {
            if (confirm('Are you sure you want to flush all cache data? This cannot be undone.')) {
                try {
                    await window.http.post('/cache/flush');
                    showSuccessInfo('Cache flushed successfully');
                    this.loadCacheData();
                } catch (error) {
                    showErrorInfo('Failed to flush cache: ' + error.message);
                }
            }
        },
        closeAddModal() {
            this.showAddModal = false;
            this.newKey = {
                key: '',
                value: '',
                ttl: 3600
            };
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
        this.loadCacheData();
    }
}