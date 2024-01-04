const exec = require('child_process').exec;

var path = require('path');

const NPM_TARGET = process.env.npm_lifecycle_event; //eslint-disable-line no-process-env

var DEV = false;
if (NPM_TARGET === 'run') {
    DEV = true;
}

const config = {
    entry: [
        './src/index.ts',
    ],
    resolve: {
        modules: [
            'src',
            'node_modules',
        ],
        extensions: ['*', '.js', '.jsx', '.ts', '.tsx'],
    },
    module: {
        rules: [
            {
                test: /\.(js|jsx|ts|tsx)$/,
                exclude: /node_modules/,
                use: {
                    loader: 'babel-loader',
                    options: {
                        cacheDirectory: true,

                        // Babel configuration is in .babelrc because jest requires it to be there.
                    },
                },
            },
            {
                test: /\.(png|jpg|gif)$/i,
                use: [
                    {
                        loader: 'url-loader',
                        options: {
                            limit: 8192,
                        },
                    },
                ],
            },
            {
                test: /\.scss$/,
                use: [
                    'style-loader',
                    {
                        loader: 'css-loader',
                    },
                    {
                        loader: 'sass-loader',
                    },
                ],
            },
        ],
    },
    externals: {
        react: 'React',
        'react-dom': 'ReactDOM',
        redux: 'Redux',
        'react-redux': 'ReactRedux',
        'prop-types': 'PropTypes',
        'react-bootstrap': 'ReactBootstrap',
    },
    output: {
        devtoolNamespace: 'jira',
        path: path.join(__dirname, '/dist'),
        publicPath: '/',
        filename: 'main.js',
    },
    devtool: 'source-map',
    plugins: [
        {
            apply: (compiler) => {
                compiler.hooks.afterEmit.tap('AfterEmitPlugin', () => {
                    exec('cd .. && make reset', (err, stdout, stderr) => {
                        if (stdout) {
                            process.stdout.write(stdout);
                        }
                        if (stderr) {
                            process.stderr.write(stderr);
                        }
                    });
                });
            },
        },
    ],
};

config.mode = 'production';

if (DEV) {
    // Development mode configuration
    config.mode = 'development';
}

// Export PRODUCTION_PERF_DEBUG=1 when running webpack to enable support for the react profiler
// even while generating production code. (Performance testing development code is typically
// not helpful.)
// See https://reactjs.org/blog/2018/09/10/introducing-the-react-profiler.html and
// https://gist.github.com/bvaughn/25e6233aeb1b4f0cdb8d8366e54a3977
if (process.env.PRODUCTION_PERF_DEBUG) { //eslint-disable-line no-process-env
    console.log('Enabling production performance debug settings'); //eslint-disable-line no-console
    config.resolve.alias['react-dom'] = 'react-dom/profiling';
    config.resolve.alias['schedule/tracing'] = 'schedule/tracing-profiling';
    config.optimization = {

        // Skip minification to make the profiled data more useful.
        minimize: false,
    };
}

module.exports = config;
