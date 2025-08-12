export default {
    name: 'AnalyticsDashboard',
    data() {
        return {
            stats: null,
            reports: [],
            loading: false,
            dateRange: {
                start: '',
                end: ''
            }
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="chart bar icon"></i>
                        Analytics Dashboard
                    </div>
                    <div class="description">
                        View detailed analytics and reports
                    </div>
                </div>
                <div class="content">
                    <div class="ui form">
                        <div class="fields">
                            <div class="field">
                                <label>Start Date</label>
                                <input type="date" v-model="dateRange.start">
                            </div>
                            <div class="field">
                                <label>End Date</label>
                                <input type="date" v-model="dateRange.end">
                            </div>
                            <div class="field">
                                <button class="ui primary button" @click="loadAnalytics" :class="{ loading: loading }">
                                    <i class="search icon"></i>
                                    Load Analytics
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="content" v-if="stats">
                    <div class="ui statistics">
                        <div class="statistic">
                            <div class="value">{{ stats.total_messages }}</div>
                            <div class="label">Total Messages</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ stats.active_users }}</div>
                            <div class="label">Active Users</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ stats.success_rate }}%</div>
                            <div class="label">Success Rate</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ stats.avg_response_time }}ms</div>
                            <div class="label">Avg Response Time</div>
                        </div>
                    </div>
                </div>
                <div class="content" v-if="reports.length > 0">
                    <h4 class="ui header">Recent Reports</h4>
                    <div class="ui relaxed divided list">
                        <div class="item" v-for="report in reports" :key="report.id">
                            <div class="right floated content">
                                <button class="ui small button" @click="downloadReport(report.id)">
                                    <i class="download icon"></i>
                                    Download
                                </button>
                            </div>
                            <div class="content">
                                <div class="header">{{ report.name }}</div>
                                <div class="description">
                                    Generated: {{ formatDate(report.created_at) }}<br>
                                    Type: {{ report.type }}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="content">
                    <button class="ui green button" @click="generateReport">
                        <i class="file alternate icon"></i>
                        Generate New Report
                    </button>
                </div>
            </div>
        </div>
    `,
    methods: {
        async loadAnalytics() {
            this.loading = true;
            try {
                const params = {};
                if (this.dateRange.start) params.start_date = this.dateRange.start;
                if (this.dateRange.end) params.end_date = this.dateRange.end;
                
                const response = await window.http.get('/analytics/realtime', { params });
                this.stats = response.data.results;
                
                const reportsResponse = await window.http.get('/analytics/daily');
                this.reports = reportsResponse.data.results || [];
                
                showSuccessInfo('Analytics loaded successfully');
            } catch (error) {
                showErrorInfo('Failed to load analytics: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        async generateReport() {
            try {
                const response = await window.http.post('/analytics/reports/generate', {
                    type: 'comprehensive',
                    start_date: this.dateRange.start,
                    end_date: this.dateRange.end
                });
                showSuccessInfo('Report generation started');
                this.loadAnalytics();
            } catch (error) {
                showErrorInfo('Failed to generate report: ' + error.message);
            }
        },
        async downloadReport(reportId) {
            try {
                const response = await window.http.get(`/analytics/reports/${reportId}/download`, {
                    responseType: 'blob'
                });
                
                const url = window.URL.createObjectURL(new Blob([response.data]));
                const link = document.createElement('a');
                link.href = url;
                link.setAttribute('download', `report_${reportId}.pdf`);
                document.body.appendChild(link);
                link.click();
                link.remove();
                
                showSuccessInfo('Report downloaded successfully');
            } catch (error) {
                showErrorInfo('Failed to download report: ' + error.message);
            }
        },
        formatDate(dateString) {
            return new Date(dateString).toLocaleString();
        }
    },
    mounted() {
        // Set default date range to last 7 days
        const end = new Date();
        const start = new Date();
        start.setDate(start.getDate() - 7);
        
        this.dateRange.start = start.toISOString().split('T')[0];
        this.dateRange.end = end.toISOString().split('T')[0];
        
        this.loadAnalytics();
    }
}