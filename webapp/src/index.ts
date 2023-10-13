import {id as pluginId} from './manifest';
import Plugin from './plugin';

window.registerPlugin(pluginId, new Plugin());
