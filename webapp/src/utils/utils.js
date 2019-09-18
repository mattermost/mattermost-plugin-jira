import {Client4} from 'mattermost-redux/client';

// Taken from mattermost-webapp/utils/utils.jsx
export function setCSRFFromCookie() {
    if (typeof document !== 'undefined' && typeof document.cookie !== 'undefined') {
        const cookies = document.cookie.split(';');
        for (let i = 0; i < cookies.length; i++) {
            const cookie = cookies[i].trim();
            if (cookie.startsWith('MMCSRF=')) {
                Client4.setCSRF(cookie.replace('MMCSRF=', ''));
                break;
            }
        }
    }
}
