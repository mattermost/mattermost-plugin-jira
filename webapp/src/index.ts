import PluginId from './plugin_id';
import Plugin from './plugin';

type WithRegisterPlugin = Window & {
    registerPlugin: (pluginId: string, plugin: Plugin) => void;
};

(window as WithRegisterPlugin).registerPlugin(PluginId, new Plugin());
