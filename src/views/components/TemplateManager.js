export default {
    name: 'TemplateManager',
    data() {
        return {
            templates: [],
            loading: false,
            newTemplate: {
                name: '',
                content: '',
                variables: [],
                category: 'general'
            },
            showCreateModal: false,
            testData: {}
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="file alternate outline icon"></i>
                        Template Manager
                    </div>
                    <div class="description">
                        Manage message templates
                    </div>
                </div>
                <div class="content">
                    <div class="ui buttons">
                        <button class="ui primary button" @click="loadTemplates" :class="{ loading: loading }">
                            <i class="refresh icon"></i>
                            Refresh
                        </button>
                        <button class="ui green button" @click="showCreateModal = true">
                            <i class="plus icon"></i>
                            Create Template
                        </button>
                    </div>
                </div>
                <div class="content" v-if="templates.length > 0">
                    <div class="ui relaxed divided list">
                        <div class="item" v-for="template in templates" :key="template.id">
                            <div class="right floated content">
                                <div class="ui buttons">
                                    <button class="ui small button" @click="testTemplate(template)">
                                        <i class="play icon"></i>
                                        Test
                                    </button>
                                    <button class="ui small button" @click="editTemplate(template)">
                                        <i class="edit icon"></i>
                                        Edit
                                    </button>
                                    <button class="ui small red button" @click="deleteTemplate(template.id)">
                                        <i class="trash icon"></i>
                                        Delete
                                    </button>
                                </div>
                            </div>
                            <div class="content">
                                <div class="header">{{ template.name }}</div>
                                <div class="description">
                                    Category: {{ template.category }}<br>
                                    Variables: {{ template.variables.join(', ') }}<br>
                                    <div class="ui small code">{{ template.content.substring(0, 100) }}...</div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <!-- Create/Edit Template Modal -->
            <div class="ui modal" :class="{ active: showCreateModal }" id="templateModal">
                <div class="header">{{ newTemplate.id ? 'Edit' : 'Create' }} Template</div>
                <div class="content">
                    <div class="ui form">
                        <div class="field">
                            <label>Template Name</label>
                            <input type="text" v-model="newTemplate.name" placeholder="Enter template name">
                        </div>
                        <div class="field">
                            <label>Category</label>
                            <select v-model="newTemplate.category" class="ui dropdown">
                                <option value="general">General</option>
                                <option value="marketing">Marketing</option>
                                <option value="support">Support</option>
                                <option value="notification">Notification</option>
                            </select>
                        </div>
                        <div class="field">
                            <label>Template Content</label>
                            <textarea v-model="newTemplate.content" placeholder="Enter template content with {{.variable}} syntax" rows="5"></textarea>
                        </div>
                        <div class="field">
                            <label>Variables (comma separated)</label>
                            <input type="text" v-model="variablesString" placeholder="name, phone, date">
                        </div>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui cancel button" @click="closeModal">Cancel</div>
                    <div class="ui primary button" @click="saveTemplate">Save</div>
                </div>
            </div>
            
            <!-- Test Template Modal -->
            <div class="ui modal" :class="{ active: showTestModal }" id="testTemplateModal">
                <div class="header">Test Template</div>
                <div class="content">
                    <div class="ui form">
                        <div class="field" v-for="variable in testTemplate.variables" :key="variable">
                            <label>{{ variable }}</label>
                            <input type="text" v-model="testData[variable]" :placeholder="'Enter ' + variable">
                        </div>
                    </div>
                    <div class="ui segment" v-if="renderedTemplate">
                        <h4>Preview:</h4>
                        <div class="ui code">{{ renderedTemplate }}</div>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui cancel button" @click="showTestModal = false">Close</div>
                    <div class="ui primary button" @click="renderTemplate">Render</div>
                </div>
            </div>
        </div>
    `,
    computed: {
        variablesString: {
            get() {
                return this.newTemplate.variables.join(', ');
            },
            set(value) {
                this.newTemplate.variables = value.split(',').map(v => v.trim()).filter(v => v);
            }
        }
    },
    data() {
        return {
            ...this.$options.data(),
            showTestModal: false,
            testTemplate: {},
            renderedTemplate: ''
        }
    },
    methods: {
        async loadTemplates() {
            this.loading = true;
            try {
                const response = await window.http.get('/templates/list');
                this.templates = response.data.results || [];
                showSuccessInfo('Templates loaded successfully');
            } catch (error) {
                showErrorInfo('Failed to load templates: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        async saveTemplate() {
            try {
                const url = this.newTemplate.id ? \`/templates/\${this.newTemplate.id}\` : '/templates/create';
                const method = this.newTemplate.id ? 'put' : 'post';
                
                await window.http[method](url, this.newTemplate);
                showSuccessInfo('Template saved successfully');
                this.closeModal();
                this.loadTemplates();
            } catch (error) {
                showErrorInfo('Failed to save template: ' + error.message);
            }
        },
        async deleteTemplate(id) {
            if (confirm('Are you sure you want to delete this template?')) {
                try {
                    await window.http.delete(\`/templates/\${id}\`);
                    showSuccessInfo('Template deleted successfully');
                    this.loadTemplates();
                } catch (error) {
                    showErrorInfo('Failed to delete template: ' + error.message);
                }
            }
        },
        editTemplate(template) {
            this.newTemplate = { ...template };
            this.showCreateModal = true;
        },
        testTemplate(template) {
            this.testTemplate = template;
            this.testData = {};
            this.renderedTemplate = '';
            this.showTestModal = true;
        },
        async renderTemplate() {
            try {
                const response = await window.http.post(\`/templates/\${this.testTemplate.id}/render\`, {
                    data: this.testData
                });
                this.renderedTemplate = response.data.results.rendered;
            } catch (error) {
                showErrorInfo('Failed to render template: ' + error.message);
            }
        },
        closeModal() {
            this.showCreateModal = false;
            this.newTemplate = {
                name: '',
                content: '',
                variables: [],
                category: 'general'
            };
        }
    },
    mounted() {
        this.loadTemplates();
    }
}