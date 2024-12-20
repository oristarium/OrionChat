class TTSDebugManager {
    constructor() {
        // Check if debug mode is enabled via URL parameter
        const urlParams = new URLSearchParams(window.location.search);
        this.debugMode = urlParams.has('debug') && (urlParams.get('debug') === '' || urlParams.get('debug') === 'true');
        
        // Log debug mode status
        console.log('Debug mode:', this.debugMode);
        console.log('ELEMENTS available:', window.ELEMENTS);
        
        // Get elements immediately
        this.elements = {
            $queueCount: window.ELEMENTS?.$debugQueueCount,
            $queueStatus: window.ELEMENTS?.$debugQueueStatus,
            $debugBox: window.ELEMENTS?.$debugBox
        };

        // Log found elements
        console.log('Found debug elements:', {
            queueCount: this.elements.$queueCount?.length,
            queueStatus: this.elements.$queueStatus?.length,
            debugBox: this.elements.$debugBox?.length,
            rawElements: this.elements
        });

        this.initialize();
    }

    initialize() {
        if (!this.debugMode) {
            console.log('Debug mode not enabled, skipping initialization');
            return;
        }

        if (!this.elements.$debugBox?.length) {
            console.error('Debug box element not found during initialization');
            return;
        }

        this.elements.$debugBox.removeClass('hidden');
        console.log('Debug mode enabled, debug box should be visible');
        
        // Re-cache elements after making debug box visible
        this.elements = {
            $queueCount: $('#queue-count'),
            $queueStatus: $('#queue-status'),
            $debugBox: $('#debug-box')
        };

        // Verify elements after re-caching
        console.log('Re-cached debug elements:', {
            queueCount: this.elements.$queueCount?.length,
            queueStatus: this.elements.$queueStatus?.length,
            debugBox: this.elements.$debugBox?.length
        });
    }

    updateQueueStatus(queue, isProcessing) {
        if (!this.debugMode) return;

        // Re-cache elements before update
        this.elements = {
            $queueCount: $('#queue-count'),
            $queueStatus: $('#queue-status'),
            $debugBox: $('#debug-box')
        };
        
        // Debug log the incoming data
        console.log('Updating queue status:', {
            queueSize: queue.size,
            isProcessing,
            hasQueueCountElement: !!this.elements.$queueCount?.length,
            hasQueueStatusElement: !!this.elements.$queueStatus?.length,
            queueItems: Array.from(queue.entries())
        });

        if (!this.elements.$queueCount?.length || !this.elements.$queueStatus?.length) {
            console.error('Debug elements not found:', {
                queueCount: this.elements.$queueCount?.length,
                queueStatus: this.elements.$queueStatus?.length
            });
            return;
        }
        
        // Update queue count
        this.elements.$queueCount.text(queue.size);
        
        // Generate queue status HTML
        const queueHTML = Array.from(queue.entries()).map(([messageId, queueItem]) => {
            const content = queueItem.message.data?.content;
            const text = content?.sanitized || content?.raw || content?.formatted || '';
            
            const status = queueItem.status || 'queued';
            const statusColor = status === 'playing' ? 'text-yellow-500' : 
                               status === 'fetching' ? 'text-blue-500' :
                               status === 'fetched' ? 'text-green-500' :
                               status === 'queued' ? 'text-gray-500' : '';
            
            return `
                <div class="grid grid-cols-2 gap-2 mb-1">
                    <div class="truncate">${text.substring(0, 30)}${text.length > 30 ? '...' : ''}</div>
                    <div class="text-right ${statusColor}">${status}</div>
                </div>
            `;
        }).join('');

        this.elements.$queueStatus.html(queueHTML);
    }

    log(...args) {
        if (this.debugMode) {
            console.log('[TTSDebug]', ...args);
        }
    }

    error(...args) {
        if (this.debugMode) {
            console.error('[TTSDebug]', ...args);
        }
    }
}

// Export the TTSDebugManager
window.TTSDebugManager = TTSDebugManager; 