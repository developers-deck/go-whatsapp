export default {
    name: 'MultiInstanceManager',
    data() {
        return {
            instances: [],
            loading: false,
            newInstance: {
                name: '',
                phone: '',
                description: ''
            },
            showCreateModal: false
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="server icon"></i>
                        Multi-Instance Manager
                    </div>
                    <div class="description">
                        Manage multiple WhatsApp instances
                    </div>
                </div>
                <div class="content">
                    <button class="ui primary button" @click="loadInstances" :class="{ loading: loading }">
                        <i class="refresh icon"></i>
                        Refresh Instances
                    </button>
                    <button class="ui green button" @click="showCreateModal = true">
                        <i class="plus icon"></i>
                        Create Instance
                    </button>
                </div>
                <div class="content" v-if="instances.length > 0">
                    <div class="ui relaxed divided list">
                        <div class="item" v-for="instance in instances" :key="instance.id">
                            <div class="right floated content">
                                <div class="ui buttons">
                                    <button class="ui small button" @click="startInstance(instance.id)" 
                                            :class="{ green: instance.status === 'stopped', disabled: instance.status === 'running' }">
                                        <i class="play icon"></i>
                                        Start
                                    </button>
                                    <button class="ui small button" @click="stopInstance(instance.id)"
                                            :class="{ red: instance.status === 'running', disabled: instance.status === 'stopped' }">
                                        <i class="stop icon"></i>
                                        Stop
                                    </button>
                                    <button class="ui small red button" @click="deleteInstance(instance.id)">
                                        <i class="trash icon"></i>
                                        Delete
                                    </button>
                                </div>
                            </div>
                            <div class="content">
                                <div class="header">{{ instance.name }}</div>
                                <div class="description">
                                    Phone: {{ instance.phone }}<br>
                                    Status: <span :class="{ 'ui green label': instance.status === 'running', 'ui red label': instance.status === 'stopped' }">
                                        {{ instance.status }}
                                    </span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <!-- Create Instance Modal -->
            <div class="ui modal" :class="{ active: showCreateModal }" id="createInstanceModal">
                <div class="header">Create New Instance</div>
                <div class="content">
                    <div class="ui form">
                        <div class="field">
                            <label>Instance Name</label>
                            <input type="text" v-model="newInstance.name" placeholder="Enter instance name">
                        </div>
                        <div class="field">
                            <label>Phone Number</label>
                            <input type="text" v-model="newInstance.phone" placeholder="Enter phone number">
                        </div>
                        <div class="field">
                            <label>Description</label>
                            <textarea v-model="newInstance.description" placeholder="Enter description"></textarea>
                        </div>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui cancel button" @click="showCreateModal = false">Cancel</div>
                    <div class="ui primary button" @click="createInstance">Create</div>
                </div>
            </div>
        </div>
    `,
    methods: {
        async loadInstances() {
            this.loading = true;
            try {
                const response = await window.http.get('/multiinstance/list');
                this.instances = response.data.results || [];
                showSuccessInfo('Instances loaded successfully');
            } catch (error) {
                showErrorInfo('Failed to load instances: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        async createInstance() {
            try {
                await window.http.post('/multiinstance/create', this.newInstance);
                showSuccessInfo('Instance created successfully');
                this.showCreateModal = false;
                this.newInstance = { name: '', phone: '', description: '' };
                this.loadInstances();
            } catch (error) {
                showErrorInfo('Failed to create instance: ' + error.message);
            }
        },
        async startInstance(id) {
            try {
                await window.http.post(\`/multiinstance/\${id}/start\`);
                showSuccessInfo('Instance started successfully');
                this.loadInstances();
            } catch (error) {
                showErrorInfo('Failed to start instance: ' + error.message);
            }
        },
        async stopInstance(id) {
            try {
                await window.http.post(\`/multiinstance/\${id}/stop\`);
                showSuccessInfo('Instance stopped successfully');
                this.loadInstances();
            } catch (error) {
                showErrorInfo('Failed to stop instance: ' + error.message);
            }
        },
        async deleteInstance(id) {
            if (confirm('Are you sure you want to delete this instance?')) {
                try {
                    await window.http.delete(\`/multiinstance/\${id}\`);
                    showSuccessInfo('Instance deleted successfully');
                    this.loadInstances();
                } catch (error) {
                    showErrorInfo('Failed to delete instance: ' + error.message);
                }
            }
        }
    },
    mounted() {
        this.loadInstances();
    }
}