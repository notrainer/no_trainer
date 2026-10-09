import autoprefixer from 'autoprefixer';
import chokidar from 'chokidar';
import cssnano from 'cssnano';
import esbuild from 'esbuild';
import fs from 'fs';
import fsExtra from 'fs-extra';
import path from 'path';
import postcss from 'postcss';
import atImport from 'postcss-import';
import tailwindcss from 'tailwindcss';
import { fileURLToPath } from 'url';

// Construct __dirname equivalent in ES Module
const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Function to check if the script was run with `--watch` or `-w`
function shouldWatch() {
    const res = process.argv.includes('--watch') || process.argv.includes('-w');
    if (res) {
        console.log('Running in watch mode...');
    }
    return res;
}

// Function to check if the script was run with `--dev` or `-D`
function isDevMode() {
    const res = process.argv.includes('--dev') || process.argv.includes('-D');
    if (res) {
        console.log('Running in development mode...');
    }
    return res;
}

const watchMode = shouldWatch();
const devMode = isDevMode();

// Path to your source CSS file
const srcCssFile = path.join(__dirname, 'assets', 'main.css');

// Path to the output CSS file in the 'dist' directory
const outputCssFile = path.join(__dirname, 'dist', 'main.css');

const distDir = path.join(__dirname, 'dist');

fs.rmSync(distDir, { recursive: true, force: true });
fs.mkdirSync(distDir, { recursive: true });

async function buildCSS() {
    try {
        const css = fs.readFileSync(srcCssFile, 'utf-8');
        const plugins = [atImport(), tailwindcss, autoprefixer];
        if (!devMode) {
            plugins.push(cssnano);
        }

        const result = await postcss(plugins)
            .process(css, { from: srcCssFile, to: outputCssFile, map: { inline: false } });

        fs.writeFileSync(outputCssFile, result.css);
        if (result.map) {
            fs.writeFileSync(`${outputCssFile}.map`, result.map.toString());
        }
    } catch (error) {
        console.error('Error building CSS:', error);
        throw error;
    }
}


const srcJsFile = path.join(__dirname, 'assets', 'main.ts');

async function buildJS() {
    await esbuild.build({
        entryPoints: [srcJsFile],
        bundle: true,
        outfile: path.join(distDir, 'main.js'),
        sourcemap: true,
        minify: true,
        define: {
            'process.env.NODE_ENV': devMode ? '"development"' : '"production"'
        },
        format: 'esm',
        platform: 'browser'
    });
}

async function copyImages() {
    const srcDir = path.join(__dirname, 'assets', 'img');
    const destDir = path.join(distDir, 'img');

    try {
        // Ensure the destination directory exists
        fsExtra.ensureDirSync(destDir);

        // Copy all files from srcDir to destDir
        await fsExtra.copy(srcDir, destDir);
        console.log('Images copied successfully!');
    } catch (error) {
        console.error('Error copying images:', error);
    }
}

console.log('Building...')
await buildJS();
await buildCSS();
// await copyImages();
console.log('Done!')

if (watchMode) {
    const jsWatcher = chokidar.watch(['assets/**/*.js', 'assets/**/*.ts'], { ignored: /(^|[\/\\])\../ });
    jsWatcher.on('change', async (file) => {
        console.log(`File ${file} has been changed, rebuilding JavaScript...`);
        await buildJS();
        console.log(`Rebuilt JavaScript...`);
    });
    console.log('Watching for JavaScript changes...');

    // Set up chokidar to watch CSS and HTML files
    const cssWatcher = chokidar.watch(['assets/**/*.css', 'templates/**/*.html'], { ignored: /(^|[\/\\])\../ });
    cssWatcher.on('change', async (file) => {
        console.log(`File ${file} has been changed, rebuilding CSS...`);
        await buildCSS();
        console.log(`Rebuilt CSS...`);
    });
    console.log('Watching for CSS changes...');

    // const imgWatcher = chokidar.watch(['assets/img/**/*'], { ignored: /(^|[\/\\])\../ });
    // imgWatcher.on('change', async (file) => {
    //     console.log(`Image file ${file} has been changed, copying images...`);
    //     await copyImages();
    //     console.log(`Copied images...`);
    // });
    // console.log('Watching for image changes...');

    const disposeOnSignal = () => {
        jsWatcher.close();
        cssWatcher.close();
        console.log('Stopped watching for changes.')
        process.exit(0); // Exit the process cleanly
    };

    // Listen for SIGINT (Ctrl+C)
    process.on('SIGINT', disposeOnSignal);
}
