export default {
    name: 'BackupManager',
    data() {
        return {
            backups: [],
            loading: false,
            backupConfig: {
                provider: 's3',
                bucket: '',
                region: '',
                access_key: '',
                secret_key: '',
                schedule: 'daily'
            },
            showConfigModal: false
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="cloud upload icon"></i>
                        Backup Manager
                    </div>
                    <div class="description">
                        Manage cloud backups and restore points
                    </div>
                </div>
                <div class="content">
                    <div class="ui buttons">
                        <button class="ui primary button" @click="loadBackups" :class="{ loading: loading }">
                            <i class="refresh icon"></i>
                            Refresh
                        </button>
                        <button class="ui green button" @click="createBackup">
                            <i class="backup icon"></i>
                            Create Backup
                        </button>
                        <button class="ui button" @click="showConfigModal = true">
                            <i class="settings icon"></i>
                            Configure
                        </button>
                    </div>
                </div>
                <div class="content" v-if="backups.length > 0">
                    <table class="ui celled table">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Size</th>
                                <th>Created</th>
                                <th>Status</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="backup in backups" :key="backup.id">
                                <td>{{ backup.name }}</td>
                                <td>{{ formatBytes(backup.size) }}</td>
                                <td>{{ formatDate(backup.created_at) }}</td>
                                <td>
                                    <span :class="{ 
                                        'ui green label': backup.status === 'completed',
                                        'ui yellow label': backup.status === 'in_progress',
                                        'ui red label': backup.status === 'failed'
                                    }">
                                        {{ backup.status }}
                                    </span>
                                </td>
                                <td>
                                    <div class="ui buttons">
                                        <button class="ui small button" @click="downloadBackup(backup.id)">
                                            <i class="download icon"></i>
                                            Download
                                        </button>
                                        <button class="ui small blue button" @click="restoreBackup(backup.id)">
                                            <i class="undo icon"></i>
                                            Restore
                                        </button>
                                        <button class="ui small red button" @click="deleteBackup(backup.id)">
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
            
            <!-- Backup Configuration Modal -->
            <div class="ui modal" :class="{ active: showConfigModal }" id="backupConfigModal">
                <div class="header">Backup Configuration</div>
                <div class="content">
                    <div class="ui form">
                        <div class="field">
                            <label>Provider</label>
                            <select v-model="backupConfig.provider" class="ui dropdown">
                                <option value="s3">Amazon S3</option>
                                <option value="gcs">Google Cloud Storage</option>
                            </select>
                        </div>
                        <div class="field">
                            <label>Bucket Name</label>
                            <input type="text" v-model="backupConfig.bucket" placeholder="Enter bucket name">
                        </div>
                        <div class="field">
                            <label>Region</label>
                            <input type="text" v-model="backupConfig.region" placeholder="Enter region">
                        </div>
                        <div class="field">
                            <label>Access Key</label>
                            <input type="text" v-model="backupConfig.access_key" placeholder="Enter access key">
                        </div>
                        <div class="field">
                            <label>Secret Key</label>
                            <input type="password" v-model="backupConfig.secret_key" placeholder="Enter secret key">
                        </div>
                        <div class="field">
                            <label>Schedule</label>
                            <select v-model="backupConfig.schedule" class="ui dropdown">
                                <option value="hourly">Hourly</option>
                                <option value="daily">Daily</option>
                                <option value="weekly">Weekly</option>
                                <option value="monthly">Monthly</option>
                            </select>
                        </div>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui cancel button" @click="showConfigModal = false">Cancel</div>
                    <div class="ui primary button" @click="saveConfig">Save Configuration</div>
                </div>
            </div>
        </div>
    `,
    methods: {
        async loadBackups() {
            this.loading = true;
            try {
                const response = await window.http.get('/backup/list');
                this.backups = response.data.results || [];
                showSuccessInfo('Backups loaded successfully');
            } catch (error) {
                showErrorInfo('Failed to load backups: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        async createBackup() {
            try {
                await window.http.post('/backup/create');
                showSuccessInfo('Backup creation started');
                this.loadBackups();
            } catch (error) {
                showErrorInfo('Failed to create backup: ' + error.message);
            }
        },
        async downloadBackup(id) {
            try {
                const response = await window.http.get(`/backup/${id}/download`, {
                    responseType: 'blob'
                });
                
                const url = window.URL.createObjectURL(new Blob([response.data]));
                const link = document.createElement('a');
                link.href = url;
                link.setAttribute('download', `backup_${id}.zip`);
                document.body.appendChild(link);
                link.click();
                link.remove();
                
                showSuccessInfo('Backup downloaded successfully');
            } catch (error) {
                showErrorInfo('Failed to download backup: ' + error.message);
            }
        },
        async restoreBackup(id) {
            if (confirm('Are you sure you want to restore this backup? This will overwrite current data.')) {
                try {
                    await window.http.post(`/backup/${id}/restore`);
                    showSuccessInfo('Backup restoration started');
                } catch (error) {
                    showErrorInfo('Failed to restore backup: ' + error.message);
                }
            }
        },
        async deleteBackup(id) {
            if (confirm('Are you sure you want to delete this backup?')) {
                try {
                    await window.http.delete(`/backup/${id}`);
                    showSuccessInfo('Backup deleted successfully');
                    this.loadBackups();
                } catch (error) {
                    showErrorInfo('Failed to delete backup: ' + error.message);
                }
            }
        },
        async saveConfig() {
            try {
                await window.http.post('/backup/config', this.backupConfig);
                showSuccessInfo('Backup configuration saved');
                this.showConfigModal = false;
            } catch (error) {
                showErrorInfo('Failed to save configuration: ' + error.message);
            }
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
        this.loadBackups();
    }
}