export default {
    name: 'QueueManager',
    data() {
        return {
            jobs: [],
            stats: null,
            loading: false,
            newJob: {
                type: 'send_message',
                priority: 'normal',
                data: {}
            },
            showCreateModal: false
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="tasks icon"></i>
                        Queue Manager
                    </div>
                    <div class="description">
                        Manage job queues and processing
                    </div>
                </div>
                <div class="content">
                    <div class="ui buttons">
                        <button class="ui primary button" @click="loadJobs" :class="{ loading: loading }">
                            <i class="refresh icon"></i>
                            Refresh
                        </button>
                        <button class="ui green button" @click="showCreateModal = true">
                            <i class="plus icon"></i>
                            Add Job
                        </button>
                        <button class="ui button" @click="pauseQueue">
                            <i class="pause icon"></i>
                            Pause Queue
                        </button>
                        <button class="ui button" @click="resumeQueue">
                            <i class="play icon"></i>
                            Resume Queue
                        </button>
                    </div>
                </div>
                <div class="content" v-if="stats">
                    <div class="ui statistics">
                        <div class="statistic">
                            <div class="value">{{ stats.pending }}</div>
                            <div class="label">Pending</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ stats.processing }}</div>
                            <div class="label">Processing</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ stats.completed }}</div>
                            <div class="label">Completed</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ stats.failed }}</div>
                            <div class="label">Failed</div>
                        </div>
                    </div>
                </div>
                <div class="content" v-if="jobs.length > 0">
                    <table class="ui celled table">
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Type</th>
                                <th>Priority</th>
                                <th>Status</th>
                                <th>Created</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="job in jobs" :key="job.id">
                                <td>{{ job.id }}</td>
                                <td>{{ job.type }}</td>
                                <td>
                                    <span :class="{ 
                                        'ui red label': job.priority === 'high',
                                        'ui yellow label': job.priority === 'normal',
                                        'ui green label': job.priority === 'low'
                                    }">
                                        {{ job.priority }}
                                    </span>
                                </td>
                                <td>
                                    <span :class="{ 
                                        'ui blue label': job.status === 'pending',
                                        'ui yellow label': job.status === 'processing',
                                        'ui green label': job.status === 'completed',
                                        'ui red label': job.status === 'failed'
                                    }">
                                        {{ job.status }}
                                    </span>
                                </td>
                                <td>{{ formatDate(job.created_at) }}</td>
                                <td>
                                    <div class="ui buttons">
                                        <button class="ui small button" @click="retryJob(job.id)" v-if="job.status === 'failed'">
                                            <i class="redo icon"></i>
                                            Retry
                                        </button>
                                        <button class="ui small red button" @click="cancelJob(job.id)" v-if="job.status === 'pending'">
                                            <i class="cancel icon"></i>
                                            Cancel
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
            
            <!-- Create Job Modal -->
            <div class="ui modal" :class="{ active: showCreateModal }" id="createJobModal">
                <div class="header">Add New Job</div>
                <div class="content">
                    <div class="ui form">
                        <div class="field">
                            <label>Job Type</label>
                            <select v-model="newJob.type" class="ui dropdown">
                                <option value="send_message">Send Message</option>
                                <option value="send_image">Send Image</option>
                                <option value="send_file">Send File</option>
                                <option value="backup">Backup</option>
                            </select>
                        </div>
                        <div class="field">
                            <label>Priority</label>
                            <select v-model="newJob.priority" class="ui dropdown">
                                <option value="high">High</option>
                                <option value="normal">Normal</option>
                                <option value="low">Low</option>
                            </select>
                        </div>
                        <div class="field">
                            <label>Job Data (JSON)</label>
                            <textarea v-model="newJob.dataJson" placeholder='{"phone": "1234567890", "message": "Hello"}'></textarea>
                        </div>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui cancel button" @click="showCreateModal = false">Cancel</div>
                    <div class="ui primary button" @click="createJob">Add Job</div>
                </div>
            </div>
        </div>
    `,
    methods: {
        async loadJobs() {
            this.loading = true;
            try {
                const response = await window.http.get('/queue/jobs');
                this.jobs = response.data.results || [];
                
                const statsResponse = await window.http.get('/queue/stats');
                this.stats = statsResponse.data.results;
                
                showSuccessInfo('Queue data loaded successfully');
            } catch (error) {
                showErrorInfo('Failed to load queue data: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        async createJob() {
            try {
                const jobData = {
                    type: this.newJob.type,
                    priority: this.newJob.priority,
                    data: JSON.parse(this.newJob.dataJson || '{}')
                };
                
                await window.http.post('/queue/jobs', jobData);
                showSuccessInfo('Job added successfully');
                this.showCreateModal = false;
                this.newJob = { type: 'send_message', priority: 'normal', dataJson: '' };
                this.loadJobs();
            } catch (error) {
                showErrorInfo('Failed to add job: ' + error.message);
            }
        },
        async retryJob(jobId) {
            try {
                await window.http.post(\`/queue/jobs/\${jobId}/retry\`);
                showSuccessInfo('Job retry initiated');
                this.loadJobs();
            } catch (error) {
                showErrorInfo('Failed to retry job: ' + error.message);
            }
        },
        async cancelJob(jobId) {
            try {
                await window.http.post(\`/queue/jobs/\${jobId}/cancel\`);
                showSuccessInfo('Job cancelled successfully');
                this.loadJobs();
            } catch (error) {
                showErrorInfo('Failed to cancel job: ' + error.message);
            }
        },
        async pauseQueue() {
            try {
                await window.http.post('/queue/pause');
                showSuccessInfo('Queue paused');
            } catch (error) {
                showErrorInfo('Failed to pause queue: ' + error.message);
            }
        },
        async resumeQueue() {
            try {
                await window.http.post('/queue/resume');
                showSuccessInfo('Queue resumed');
            } catch (error) {
                showErrorInfo('Failed to resume queue: ' + error.message);
            }
        },
        formatDate(dateString) {
            return new Date(dateString).toLocaleString();
        }
    },
    mounted() {
        this.loadJobs();
    }
}