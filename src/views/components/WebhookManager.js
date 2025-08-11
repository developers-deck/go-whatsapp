export default {
    name: 'WebhookManager',
    data() {
        return {
            webhooks: [],
            deliveries: [],
            loading: false,
            newWebhook: {
                name: '',
                url: '',
                events: [],
                secret: '',
                active: true
            },
            showCreateModal: false,
            availableEvents: [
                'message.received',
                'message.sent',
                'user.login',
                'user.logout',
                'group.created',
                'group.updated'
            ]
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="webhook icon"></i>
                        Webhook Manager
                    </div>
                    <div class="description">
                        Manage webhook endpoints and deliveries
                    </div>
                </div>
                <div class="content">
                    <div class="ui buttons">
                        <button class="ui primary button" @click="loadWebhooks" :class="{ loading: loading }">
                            <i class="refresh icon"></i>
                            Refresh
                        </button>
                        <button class="ui green button" @click="showCreateModal = true">
                            <i class="plus icon"></i>
                            Create Webhook
                        </button>
                    </div>
                </div>
                <div class="content" v-if="webhooks.length > 0">
                    <div class="ui relaxed divided list">
                        <div class="item" v-for="webhook in webhooks" :key="webhook.id">
                            <div class="right floated content">
                                <div class="ui buttons">
                                    <button class="ui small button" @click="testWebhook(webhook.id)">
                                        <i class="play icon"></i>
                                        Test
                                    </button>
                                    <button class="ui small button" @click="toggleWebhook(webhook.id, !webhook.active)"
                                            :class="{ green: !webhook.active, red: webhook.active }">
                                        <i :class="webhook.active ? 'pause icon' : 'play icon'"></i>
                                        {{ webhook.active ? 'Disable' : 'Enable' }}
                                    </button>
                                    <button class="ui small button" @click="editWebhook(webhook)">
                                        <i class="edit icon"></i>
                                        Edit
                                    </button>
                                    <button class="ui small red button" @click="deleteWebhook(webhook.id)">
                                        <i class="trash icon"></i>
                                        Delete
                                    </button>
                                </div>
                            </div>
                            <div class="content">
                                <div class="header">{{ webhook.name }}</div>
                                <div class="description">
                                    URL: {{ webhook.url }}<br>
                                    Events: {{ webhook.events.join(', ') }}<br>
                                    Status: <span :class="{ 'ui green label': webhook.active, 'ui red label': !webhook.active }">
                                        {{ webhook.active ? 'Active' : 'Inactive' }}
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="content">
                    <h4 class="ui header">Recent Deliveries</h4>
                    <div class="ui relaxed divided list" v-if="deliveries.length > 0">
                        <div class="item" v-for="delivery in deliveries" :key="delivery.id">
                            <div class="right floated content">
                                <span :class="{ 
                                    'ui green label': delivery.status === 'success',
                                    'ui red label': delivery.status === 'failed',
                                    'ui yellow label': delivery.status === 'pending'
                                }">
                                    {{ delivery.status }}
                                </span>
                            </div>
                            <div class="content">
                                <div class="header">{{ delivery.event }}</div>
                                <div class="description">
                                    URL: {{ delivery.url }}<br>
                                    Attempts: {{ delivery.attempts }}<br>
                                    Last Attempt: {{ formatDate(delivery.last_attempt) }}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <!-- Create/Edit Webhook Modal -->
            <div class="ui modal" :class="{ active: showCreateModal }" id="webhookModal">
                <div class="header">{{ newWebhook.id ? 'Edit' : 'Create' }} Webhook</div>
                <div class="content">
                    <div class="ui form">
                        <div class="field">
                            <label>Webhook Name</label>
                            <input type="text" v-model="newWebhook.name" placeholder="Enter webhook name">
                        </div>
                        <div class="field">
                            <label>URL</label>
                            <input type="url" v-model="newWebhook.url" placeholder="https://example.com/webhook">
                        </div>
                        <div class="field">
                            <label>Secret (optional)</label>
                            <input type="text" v-model="newWebhook.secret" placeholder="Enter secret for signature verification">
                        </div>
                        <div class="field">
                            <label>Events</label>
                            <div class="ui multiple selection dropdown" ref="eventsDropdown">
                                <input type="hidden">
                                <div class="default text">Select events</div>
                                <div class="menu">
                                    <div class="item" v-for="event in availableEvents" :key="event" :data-value="event">
                                        {{ event }}
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="field">
                            <div class="ui checkbox">
                                <input type="checkbox" v-model="newWebhook.active">
                                <label>Active</label>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui cancel button" @click="closeModal">Cancel</div>
                    <div class="ui primary button" @click="saveWebhook">Save</div>
                </div>
            </div>
        </div>
    `,
    methods: {
        async loadWebhooks() {
            this.loading = true;
            try {
                const response = await window.http.get('/webhook/list');
                this.webhooks = response.data.results || [];
                
                const deliveriesResponse = await window.http.get('/webhook/deliveries');
                this.deliveries = deliveriesResponse.data.results || [];
                
                showSuccessInfo('Webhooks loaded successfully');
            } catch (error) {
                showErrorInfo('Failed to load webhooks: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        async saveWebhook() {
            try {
                const url = this.newWebhook.id ? \`/webhook/\${this.newWebhook.id}\` : '/webhook/create';
                const method = this.newWebhook.id ? 'put' : 'post';
                
                await window.http[method](url, this.newWebhook);
                showSuccessInfo('Webhook saved successfully');
                this.closeModal();
                this.loadWebhooks();
            } catch (error) {
                showErrorInfo('Failed to save webhook: ' + error.message);
            }
        },
        async deleteWebhook(id) {
            if (confirm('Are you sure you want to delete this webhook?')) {
                try {
                    await window.http.delete(\`/webhook/\${id}\`);
                    showSuccessInfo('Webhook deleted successfully');
                    this.loadWebhooks();
                } catch (error) {
                    showErrorInfo('Failed to delete webhook: ' + error.message);
                }
            }
        },
        async testWebhook(id) {
            try {
                await window.http.post(\`/webhook/\${id}/test\`);
                showSuccessInfo('Test webhook sent');
            } catch (error) {
                showErrorInfo('Failed to test webhook: ' + error.message);
            }
        },
        async toggleWebhook(id, active) {
            try {
                await window.http.patch(\`/webhook/\${id}\`, { active });
                showSuccessInfo(\`Webhook \${active ? 'enabled' : 'disabled'}\`);
                this.loadWebhooks();
            } catch (error) {
                showErrorInfo('Failed to toggle webhook: ' + error.message);
            }
        },
        editWebhook(webhook) {
            this.newWebhook = { ...webhook };
            this.showCreateModal = true;
            this.$nextTick(() => {
                $(this.$refs.eventsDropdown).dropdown('set selected', webhook.events);
            });
        },
        closeModal() {
            this.showCreateModal = false;
            this.newWebhook = {
                name: '',
                url: '',
                events: [],
                secret: '',
                active: true
            };
        },
        formatDate(dateString) {
            return new Date(dateString).toLocaleString();
        }
    },
    mounted() {
        this.loadWebhooks();
        
        // Initialize Semantic UI dropdown
        this.$nextTick(() => {
            $(this.$refs.eventsDropdown).dropdown({
                onChange: (value) => {
                    this.newWebhook.events = value ? value.split(',') : [];
                }
            });
        });
    }
}