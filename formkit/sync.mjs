// Build the shared player from the user's FormKit checkout and copy its assets.
import { execFileSync } from 'node:child_process';
import { cp, mkdir, mkdtemp, readFile, readdir, realpath, rm, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const source = await realpath(resolve(process.argv[2] ?? join(root, '../formkit')));
const destination = join(root, 'assets/formkit');
const work = await mkdtemp(join(tmpdir(), 'groove-formkit-'));
const require = createRequire(join(source, 'package.json'));
const git = (...args) => execFileSync('git', ['-C', source, ...args], {encoding: 'utf8'}).trim();

try {
  if (git('status', '--porcelain', '--untracked-files=no')) {
    throw new Error('Use a FormKit checkout without tracked edits for a reproducible bundle.');
  }
  const revision = git('rev-parse', 'HEAD');
  const { build } = await import(pathToFileURL(require.resolve('vite')));
  let engravingAdapted = false;
  await build({
    root: source,
    configFile: false,
    publicDir: false,
    plugins: [{
      name: 'groove-lab-written-beam-groups',
      enforce: 'pre',
      transform(code, id) {
        if (id !== join(source, 'src/components/grooves/GrooveScore.tsx')) return;
        const changes = [
          ['  engraveDrums,\n', ''],
          ["import { GLYPH } from '../music-glyphs.ts';",
           "import { GLYPH } from '../music-glyphs.ts';\nimport { engraveLoopDrums } from '@groove-lab-beams';"],
          ['const { beats, feet } = engraveDrums(events, beatTicks);',
           'const { beats, feet, beatStartAt, unflagged } = engraveLoopDrums(events, beatTicks, barTicks, (song as Song & { drumBeamGroups?: number[] }).drumBeamGroups);'],
          ['column.display >= beatTicks / (beatTicks === 1440 ? 1 : 1) &&\n            column.at % beatTicks === 0;',
           'unflagged(column);'],
          ['const beatStart = Math.floor(c.at / beatTicks) * beatTicks;\n          if (!(c.at === beatStart && c.display >= beatTicks)) {',
           'const beatStart = beatStartAt(c.at);\n          if (!unflagged(c)) {'],
        ];
        for (const [original, replacement] of changes) {
          if (code.split(original).length !== 2) {
            throw new Error('FormKit drum engraving changed; update the beam-group adapter.');
          }
          code = code.replace(original, replacement);
        }
        engravingAdapted = true;
        return {code, map:null};
      },
    }],

    resolve: {
      alias: [
        {find: '@formkit', replacement: join(source, 'src')},
        {find: '@groove-lab-beams', replacement: join(root, 'formkit/drum-beams.ts')},
        ...['react', 'react/jsx-runtime', 'react-dom', 'react-dom/client'].map(name => ({
          find: new RegExp(`^${name}$`), replacement: require.resolve(name),
        })),
      ],
    },
    define: { "process.env.NODE_ENV": JSON.stringify("production") },
    oxc: { jsx: { runtime: 'automatic' } },
    build: {
      target: 'es2022',
      outDir: work,
      emptyOutDir: true,
      lib: {
        entry: {
          formkit: join(root, 'formkit/adapter.tsx'),
          'groove-worklet': join(source, 'src/music/groove.worklet.ts'),
        },
        formats: ['es'],
      },
      rollupOptions: {
        output: {
          entryFileNames: '[name].js',
          chunkFileNames: '[name].js',
        },
      },
    },
  });
  if (!engravingAdapted) throw new Error('The FormKit score renderer did not receive the beam-group adapter.');
  for (const file of await readdir(work)) {
    if (!file.endsWith('.js')) continue;
    const path = join(work, file);
    const js = await readFile(path, 'utf8');
    // Rolldown adds source-region comments with checkout-relative paths.
    await writeFile(path, js.replace(/^[ \t]*\/\/#(?:end)?region.*(?:\r?\n|$)/gm, ''));
  }
  const css = await readFile(join(source, 'src/grooves.css'), 'utf8');
  const section = name => css.indexOf(`/* ------------------------------------------------------------- ${name} */`);
  const part = (first, last) => {
    const start = section(first);
    const end = section(last);
    if (start < 0 || end <= start) throw new Error(`Missing FormKit CSS sections ${first} and ${last}`);
    return css.slice(start, end);
  };
  const font = '@font-face {font-family: Inter; src: url("./inter-latin-W-normal.woff2") format("woff2"); font-display: swap; font-weight: W;}';
  const shared = css.slice(0, section('library')) + part('library', 'performance') + part('score', 'chips, levels');
  const appCSS = await readFile(join(source, 'src/style.css'), 'utf8');
  const base = appCSS.slice(0, appCSS.indexOf('.instrument {'));
  const settings = await readFile(join(source, 'src/app-settings.css'), 'utf8');
  const light = settings.match(/:root\[data-theme='light'\] \{[^}]+\}\s*:root\[data-theme='light'\] body \{[^}]+\}/)?.[0];
  if (!light) throw new Error('Missing FormKit light-theme styles');
  await writeFile(join(work, 'theme.css'), [
    `/* FormKit theme and notation styles at ${revision}. */`,
    ...[400, 500, 600, 700].map(weight => font.replaceAll('W', String(weight))),
    base, light,
    shared.replace('./assets/fonts/BopkitMusic.woff', './BopkitMusic.woff'),
  ].join('\n').trimEnd() + '\n');

  await cp(join(source, 'src/assets/fonts/BopkitMusic.woff'), join(work, 'BopkitMusic.woff'));
  const musicLicense = await readFile(join(source, 'src/assets/fonts/BopkitMusic-LICENSE.txt'), 'utf8');
  await writeFile(join(work, 'BopkitMusic-LICENSE.txt'), musicLicense.replace(/[ \t]+$/gm, ''));
  for (const weight of [400, 500, 600, 700]) {
    const file = `inter-latin-${weight}-normal.woff2`;
    await cp(require.resolve(`@fontsource/inter/files/${file}`), join(work, file));
  }
  await cp(join(source, 'public/Inter-LICENSE.txt'), join(work, 'Inter-LICENSE.txt'));
  for (const name of ['react', 'react-dom', 'scheduler']) {
    const from = require.resolve(`${name}/package.json`);
    await cp(join(dirname(from), 'LICENSE'), join(work, `${name}-LICENSE.txt`));
  }
  await cp(join(source, 'public/audio'), join(work, 'audio'), {recursive:true});
  const credits = join(work, 'audio/CREDITS.md');
  await writeFile(credits, (await readFile(credits, 'utf8')).replaceAll(
    /\/Users\/[^/]+\/workspace\/formkit\/public\/audio\//g, 'assets/formkit/audio/',
  ));
  await writeFile(join(work, 'SOURCE.txt'), `FormKit / BopKit commit ${revision}\n`);
  // Commit the generated directory so the Go app can run on its own.
  await mkdir(destination, {recursive:true});
  await cp(work, destination, {recursive:true,force:true});
} finally {
  await rm(work, {recursive:true,force:true});
}
