// API Configuration
const API_BASE_URL = 'http://localhost:8080';

// DOM Elements
const shortenForm = document.getElementById('shortenForm');
const resultSection = document.getElementById('resultSection');
const loadingOverlay = document.getElementById('loadingOverlay');
const toast = document.getElementById('toast');

// Form submission handler
shortenForm.addEventListener('submit', async (e) => {
    e.preventDefault();

    const originalUrl = document.getElementById('originalUrl').value;
    const customAlias = document.getElementById('customAlias').value;
    const expiresIn = document.getElementById('expiresIn').value;

    // Validate URL
    if (!isValidUrl(originalUrl)) {
        showToast('Please enter a valid URL', 'error');
        return;
    }

    // Show loading
    showLoading();

    try {
        // Prepare request body
        const requestBody = {
            url: originalUrl
        };

        if (customAlias) {
            requestBody.custom_alias = customAlias;
        }

        if (expiresIn) {
            requestBody.expires_in_hours = parseInt(expiresIn);
        }

        // Make API request
        const response = await fetch(`${API_BASE_URL}/api/v1/urls`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(requestBody)
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || 'Failed to create short URL');
        }

        // Display result
        displayResult(data.data);
        showToast('Short URL created successfully!', 'success');

    } catch (error) {
        console.error('Error:', error);
        showToast(error.message || 'Failed to create short URL', 'error');
    } finally {
        hideLoading();
    }
});

// Display result
function displayResult(data) {
    // Populate result fields
    document.getElementById('shortUrl').value = data.short_url;
    document.getElementById('originalUrlDisplay').textContent = truncateUrl(data.original_url, 50);
    document.getElementById('createdAt').textContent = formatDate(data.created_at);

    // Show/hide expiration if present
    if (data.expires_at) {
        document.getElementById('expiresAtStat').style.display = 'flex';
        document.getElementById('expiresAt').textContent = formatDate(data.expires_at);
    } else {
        document.getElementById('expiresAtStat').style.display = 'none';
    }

    // Hide form, show result
    shortenForm.style.display = 'none';
    resultSection.classList.remove('hidden');
}

// Reset form
function resetForm() {
    shortenForm.reset();
    shortenForm.style.display = 'block';
    resultSection.classList.add('hidden');
}

// Copy to clipboard
function copyToClipboard() {
    const shortUrlInput = document.getElementById('shortUrl');
    shortUrlInput.select();
    shortUrlInput.setSelectionRange(0, 99999); // For mobile devices

    navigator.clipboard.writeText(shortUrlInput.value).then(() => {
        showToast('Copied to clipboard!', 'success');
    }).catch(err => {
        console.error('Failed to copy:', err);
        showToast('Failed to copy to clipboard', 'error');
    });
}

// Show loading overlay
function showLoading() {
    loadingOverlay.classList.remove('hidden');
}

// Hide loading overlay
function hideLoading() {
    loadingOverlay.classList.add('hidden');
}

// Show toast notification
function showToast(message, type = 'success') {
    const toastMessage = document.getElementById('toastMessage');
    toastMessage.textContent = message;

    // Update toast color based on type
    if (type === 'error') {
        toast.style.background = '#ef4444';
    } else if (type === 'warning') {
        toast.style.background = '#f59e0b';
    } else {
        toast.style.background = '#10b981';
    }

    toast.classList.remove('hidden');

    // Auto-hide after 3 seconds
    setTimeout(() => {
        toast.classList.add('hidden');
    }, 3000);
}

// Validate URL
function isValidUrl(string) {
    try {
        const url = new URL(string);
        return url.protocol === 'http:' || url.protocol === 'https:';
    } catch (_) {
        return false;
    }
}

// Format date
function formatDate(dateString) {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = date - now;
    const diffMins = Math.round(diffMs / 60000);
    const diffHours = Math.round(diffMs / 3600000);
    const diffDays = Math.round(diffMs / 86400000);

    // If in the past
    if (diffMs < 0) {
        const absDiffMins = Math.abs(diffMins);
        const absDiffHours = Math.abs(diffHours);
        const absDiffDays = Math.abs(diffDays);

        if (absDiffMins < 60) {
            return `${absDiffMins} minute${absDiffMins !== 1 ? 's' : ''} ago`;
        } else if (absDiffHours < 24) {
            return `${absDiffHours} hour${absDiffHours !== 1 ? 's' : ''} ago`;
        } else if (absDiffDays < 7) {
            return `${absDiffDays} day${absDiffDays !== 1 ? 's' : ''} ago`;
        }
    } else {
        // If in the future
        if (diffMins < 60) {
            return `in ${diffMins} minute${diffMins !== 1 ? 's' : ''}`;
        } else if (diffHours < 24) {
            return `in ${diffHours} hour${diffHours !== 1 ? 's' : ''}`;
        } else if (diffDays < 7) {
            return `in ${diffDays} day${diffDays !== 1 ? 's' : ''}`;
        }
    }

    // Default format
    return date.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// Truncate URL
function truncateUrl(url, maxLength) {
    if (url.length <= maxLength) return url;
    return url.substring(0, maxLength) + '...';
}

// Navigation handling
document.querySelectorAll('.nav-link').forEach(link => {
    link.addEventListener('click', (e) => {
        e.preventDefault();

        // Remove active class from all links
        document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));

        // Add active class to clicked link
        link.classList.add('active');

        // Handle navigation (for future dashboard/analytics pages)
        const target = link.getAttribute('href').substring(1);
        console.log('Navigate to:', target);

        // For now, just show a toast
        if (target === 'dashboard') {
            showToast('Dashboard coming soon!', 'warning');
        } else if (target === 'analytics') {
            showToast('Analytics coming soon!', 'warning');
        }
    });
});

// Add smooth scroll behavior
document.documentElement.style.scrollBehavior = 'smooth';

// Add input validation feedback
const urlInput = document.getElementById('originalUrl');
urlInput.addEventListener('blur', () => {
    if (urlInput.value && !isValidUrl(urlInput.value)) {
        urlInput.style.borderColor = '#ef4444';
        showToast('Please enter a valid URL starting with http:// or https://', 'error');
    } else {
        urlInput.style.borderColor = '';
    }
});

// Add custom alias validation
const aliasInput = document.getElementById('customAlias');
aliasInput.addEventListener('input', () => {
    const value = aliasInput.value;
    const isValid = /^[a-zA-Z0-9_-]*$/.test(value);

    if (value && !isValid) {
        aliasInput.style.borderColor = '#ef4444';
    } else {
        aliasInput.style.borderColor = '';
    }
});

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    // Cmd/Ctrl + K to focus URL input
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        urlInput.focus();
    }

    // Escape to reset form
    if (e.key === 'Escape' && !resultSection.classList.contains('hidden')) {
        resetForm();
    }
});

console.log('🚀 URL Shortener UI loaded successfully!');
