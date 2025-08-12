export default {
    name: 'FileManagerDashboard',
    data() {
        return {
            files: [],
            stats: null,
            loading: false,
            uploadProgress: 0,
            showUploadModal: false,
            selectedFile: null,
            fileCategory: 'other',
            showPreviewModal: false,
            previewFile: null
        }
    },
    template: `
        <div class="column">
            <div class="ui card">
                <div class="content">
                    <div class="header">
                        <i class="folder open icon"></i>
                        File Manager
                    </div>
                    <div class="description">
                        Manage uploaded files and media
                    </div>
                </div>
                <div class="content">
                    <div class="ui buttons">
                        <button class="ui primary button" @click="loadFiles" :class="{ loading: loading }">
                            <i class="refresh icon"></i>
                            Refresh
                        </button>
                        <button class="ui green button" @click="showUploadModal = true">
                            <i class="upload icon"></i>
                            Upload File
                        </button>
                    </div>
                </div>
                <div class="content" v-if="stats">
                    <div class="ui statistics">
                        <div class="statistic">
                            <div class="value">{{ stats.total_files }}</div>
                            <div class="label">Total Files</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ formatBytes(stats.total_size) }}</div>
                            <div class="label">Total Size</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ stats.images }}</div>
                            <div class="label">Images</div>
                        </div>
                        <div class="statistic">
                            <div class="value">{{ stats.documents }}</div>
                            <div class="label">Documents</div>
                        </div>
                    </div>
                </div>
                <div class="content" v-if="files.length > 0">
                    <table class="ui celled table">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Type</th>
                                <th>Size</th>
                                <th>Uploaded</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="file in files" :key="file.id">
                                <td>
                                    <i :class="getFileIcon(file.type)"></i>
                                    {{ file.name }}
                                </td>
                                <td>{{ file.type }}</td>
                                <td>{{ formatBytes(file.size) }}</td>
                                <td>{{ formatDate(file.uploaded_at) }}</td>
                                <td>
                                    <div class="ui buttons">
                                        <button class="ui small button" @click="downloadFile(file.id)">
                                            <i class="download icon"></i>
                                            Download
                                        </button>
                                        <button class="ui small button" @click="previewFile(file)" v-if="isPreviewable(file.type)">
                                            <i class="eye icon"></i>
                                            Preview
                                        </button>
                                        <button class="ui small red button" @click="deleteFile(file.id)">
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
            
            <!-- Upload Modal -->
            <div class="ui modal" :class="{ active: showUploadModal }" id="uploadModal">
                <div class="header">Upload File</div>
                <div class="content">
                    <div class="ui form">
                        <div class="field">
                            <label>Select File</label>
                            <input type="file" @change="handleFileSelect" ref="fileInput">
                        </div>
                        <div class="field" v-if="selectedFile">
                            <label>Category</label>
                            <select v-model="fileCategory" class="ui dropdown">
                                <option value="image">Image</option>
                                <option value="document">Document</option>
                                <option value="audio">Audio</option>
                                <option value="video">Video</option>
                                <option value="other">Other</option>
                            </select>
                        </div>
                        <div class="ui progress" v-if="uploadProgress > 0" :class="{ success: uploadProgress === 100 }">
                            <div class="bar" :style="{ width: uploadProgress + '%' }">
                                <div class="progress">{{ uploadProgress }}%</div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui cancel button" @click="closeUploadModal">Cancel</div>
                    <div class="ui primary button" @click="uploadFile" :class="{ loading: uploadProgress > 0 && uploadProgress < 100 }">
                        Upload
                    </div>
                </div>
            </div>
            
            <!-- Preview Modal -->
            <div class="ui modal" :class="{ active: showPreviewModal }" id="previewModal">
                <div class="header">File Preview</div>
                <div class="content">
                    <div class="ui center aligned container" v-if="previewFile">
                        <img v-if="previewFile.type.startsWith('image/')" :src="previewFile.url" style="max-width: 100%; max-height: 500px;">
                        <video v-else-if="previewFile.type.startsWith('video/')" :src="previewFile.url" controls style="max-width: 100%; max-height: 500px;"></video>
                        <audio v-else-if="previewFile.type.startsWith('audio/')" :src="previewFile.url" controls></audio>
                    </div>
                </div>
                <div class="actions">
                    <div class="ui button" @click="showPreviewModal = false">Close</div>
                </div>
            </div>
        </div>
    `,
    methods: {
        async loadFiles() {
            this.loading = true;
            try {
                const response = await window.http.get('/files/list');
                this.files = response.data.results || [];
                
                const statsResponse = await window.http.get('/files/stats');
                this.stats = statsResponse.data.results;
                
                showSuccessInfo('Files loaded successfully');
            } catch (error) {
                showErrorInfo('Failed to load files: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        handleFileSelect(event) {
            this.selectedFile = event.target.files[0];
            if (this.selectedFile) {
                // Auto-detect category based on file type
                if (this.selectedFile.type.startsWith('image/')) {
                    this.fileCategory = 'image';
                } else if (this.selectedFile.type.startsWith('video/')) {
                    this.fileCategory = 'video';
                } else if (this.selectedFile.type.startsWith('audio/')) {
                    this.fileCategory = 'audio';
                } else if (this.selectedFile.type.includes('pdf') || this.selectedFile.type.includes('document')) {
                    this.fileCategory = 'document';
                } else {
                    this.fileCategory = 'other';
                }
            }
        },
        async uploadFile() {
            if (!this.selectedFile) {
                showErrorInfo('Please select a file');
                return;
            }
            
            const formData = new FormData();
            formData.append('file', this.selectedFile);
            formData.append('category', this.fileCategory);
            
            try {
                await window.http.post('/filemanager/upload', formData, {
                    headers: {
                        'Content-Type': 'multipart/form-data'
                    },
                    onUploadProgress: (progressEvent) => {
                        this.uploadProgress = Math.round((progressEvent.loaded * 100) / progressEvent.total);
                    }
                });
                
                showSuccessInfo('File uploaded successfully');
                this.closeUploadModal();
                this.loadFiles();
            } catch (error) {
                showErrorInfo('Failed to upload file: ' + error.message);
                this.uploadProgress = 0;
            }
        },
        async downloadFile(fileId) {
            try {
                const response = await window.http.get(`/filemanager/download/${fileId}`, {
                    responseType: 'blob'
                });
                
                const url = window.URL.createObjectURL(new Blob([response.data]));
                const link = document.createElement('a');
                link.href = url;
                link.setAttribute('download', `file_${fileId}`);
                document.body.appendChild(link);
                link.click();
                link.remove();
                
                showSuccessInfo('File downloaded successfully');
            } catch (error) {
                showErrorInfo('Failed to download file: ' + error.message);
            }
        },
        async deleteFile(fileId) {
            if (confirm('Are you sure you want to delete this file?')) {
                try {
                    await window.http.delete(`/filemanager/${fileId}`);
                    showSuccessInfo('File deleted successfully');
                    this.loadFiles();
                } catch (error) {
                    showErrorInfo('Failed to delete file: ' + error.message);
                }
            }
        },
        previewFile(file) {
            this.previewFile = {
                ...file,
                url: `/filemanager/download/${file.id}`
            };
            this.showPreviewModal = true;
        },
        closeUploadModal() {
            this.showUploadModal = false;
            this.selectedFile = null;
            this.uploadProgress = 0;
            this.$refs.fileInput.value = '';
        },
        getFileIcon(type) {
            if (type.startsWith('image/')) return 'file image icon';
            if (type.startsWith('video/')) return 'file video icon';
            if (type.startsWith('audio/')) return 'file audio icon';
            if (type.includes('pdf')) return 'file pdf icon';
            if (type.includes('word')) return 'file word icon';
            if (type.includes('excel')) return 'file excel icon';
            return 'file icon';
        },
        isPreviewable(type) {
            return type.startsWith('image/') || type.startsWith('video/') || type.startsWith('audio/');
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
        this.loadFiles();
    }
}