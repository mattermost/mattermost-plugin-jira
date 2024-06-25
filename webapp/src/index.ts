import manifest from './manifest';
import Plugin from './plugin';

window.registerPlugin(manifest.id, new Plugin());
