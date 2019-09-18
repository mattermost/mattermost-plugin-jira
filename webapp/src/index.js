import PluginId from './plugin_id';
import Plugin from './plugin.jsx';
import {setCSRFFromCookie} from './utils/utils';

setCSRFFromCookie();
window.registerPlugin(PluginId, new Plugin());
