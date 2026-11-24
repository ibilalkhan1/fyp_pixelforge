'use strict';
const $ = id => document.getElementById(id);
const canvas = $('game'), ctx = canvas.getContext('2d'), sprite = $('sprite'), sx = sprite.getContext('2d');
let scene, playing = false, tool = 'wall', paint = 11, palette = [], ready = false, lastTick = 0, wasmBytes;
const keys = new Set();
let touch = [0, 0];
const defaultSprite = [0,0,11,11,11,11,0,0,0,11,11,11,11,11,11,0,11,11,7,0,11,7,0,11,11,11,7,0,11,7,0,11,11,11,11,11,11,11,11,11,0,11,11,10,10,11,11,0,0,11,11,11,11,11,11,0,0,11,0,0,0,0,11,0];
function project(kind = 'collect') {
  const walls = [];
  for (let x = 0; x < 32; x++) walls.push([x, 0], [x, 17]);
  for (let y = 1; y < 17; y++) walls.push([0, y], [31, y]);
  const s = {version:1, name:'My first game', mode:'escape', goal:[28,9], walls, gems:[], hazards:[], spawn:[3,9], color:11, speed:1.5, sprite:[...defaultSprite]};
  if (kind === 'collect') {
    s.name = 'Crystal courtyard'; s.mode = 'collect'; s.goal = null;
    s.gems = [[6,4],[15,13],[25,4],[27,13]];
    for (let y = 3; y < 14; y++) if (y !== 8) walls.push([10,y]);
    for (let x = 16; x < 28; x++) if (x !== 21) walls.push([x,7]);
  } else if (kind === 'escape') {
    s.name = 'Escape the red maze'; s.gems = [[7,9],[18,5]];
    s.hazards = [[14,9],[24,13]];
    for (let y = 1; y < 17; y++) {
      if (y !== 4) walls.push([10,y]);
      if (y !== 13) walls.push([21,y]);
    }
  }
  return s;
}
function announce(message) { $('status').textContent = message; }
function objective() {
  if (scene.mode === 'escape') return 'Reach the green finish. Avoid red hazards. Crystals are a bonus.';
  return scene.gems.length ? 'Collect every crystal. Avoid red hazards.' : 'Add crystals to create your objective, or choose Reach the finish.';
}
function setPlaying(value) {
  if (!ready) return;
  playing = value; keys.clear(); touch = [0,0];
  $('play').textContent = playing ? 'Pause editing' : 'Play scene';
  if (playing) canvas.focus();
  announce(playing ? 'Playing · ' + (scene.mode === 'escape' ? 'reach the finish' : 'collect every crystal') : 'Paused · click a tile to edit');
}
function apply(next = scene) {
  if (!ready) return;
  const candidate = structuredClone(next);
  candidate.mode ||= 'collect'; candidate.hazards ||= [];
  const error = window.pfLoad(JSON.stringify(candidate));
  if (error) throw Error(error);
  scene = candidate;
  $('name').value = scene.name; $('speed').value = scene.speed; $('speedValue').value = scene.speed; $('mode').value = scene.mode;
  drawSprite(); frame(false);
}
function frame(advance) {
  if (!ready) return;
  const dx = touch[0] || +(keys.has('ArrowRight') || keys.has('d')) - +(keys.has('ArrowLeft') || keys.has('a'));
  const dy = touch[1] || +(keys.has('ArrowDown') || keys.has('s')) - +(keys.has('ArrowUp') || keys.has('w'));
  ctx.putImageData(new ImageData(window.pfTick(advance && playing, dx, dy, $('grid').checked), 320, 180), 0, 0);
  const st = JSON.parse(window.pfState());
  $('stats').textContent = `${st.score}/${st.total} crystals · tick ${st.tick} · position ${st.x}, ${st.y}`;
  for (const field of ['tick','x','y','score','won','lost']) canvas.dataset[field] = String(st[field]);
  $('objective').textContent = st.lost ? 'You hit a hazard. Reset to try again, or pause to edit.' : st.won ? 'Level complete! Reset to play again, or pause to edit.' : objective();
  if (st.won) announce('Level complete');
  if (st.lost) announce('Game over · hit Reset to try again');
}
function loop(t) {
  if (t - lastTick >= 1000 / 30) { frame(true); lastTick = t; }
  requestAnimationFrame(loop);
}
function drawSprite() {
  sx.clearRect(0,0,128,128);
  scene.sprite.forEach((c,i) => {
    sx.fillStyle = palette[c]; sx.fillRect(i%8*16, Math.floor(i/8)*16,16,16);
    sx.strokeStyle = '#ffffff20'; sx.strokeRect(i%8*16, Math.floor(i/8)*16,16,16);
  });
}
function coordinate(event, target, cols, rows) {
  const r = target.getBoundingClientRect();
  return [Math.max(0, Math.min(cols-1, Math.floor((event.clientX-r.left)/r.width*cols))), Math.max(0, Math.min(rows-1, Math.floor((event.clientY-r.top)/r.height*rows)))];
}
canvas.addEventListener('pointerdown', event => {
  if (!ready || playing) return;
  const next = structuredClone(scene), cell = coordinate(event, canvas, 32,18), same = c => c && c[0] === cell[0] && c[1] === cell[1];
  if (same(next.spawn) && tool !== 'spawn') { announce('Move the player before changing its tile'); return; }
  if (same(next.goal) && tool !== 'goal') { announce('Move the finish before changing its tile'); return; }
  if (tool === 'goal' && same(next.spawn)) { announce('Place the finish away from the player'); return; }
  if (tool === 'gem' && next.gems.length >= 100 && !next.gems.some(same)) { announce('Maximum 100 crystals'); return; }
  if (tool === 'hazard' && next.hazards.length >= 100 && !next.hazards.some(same)) { announce('Maximum 100 hazards'); return; }
  next.walls = next.walls.filter(c => !same(c)); next.gems = next.gems.filter(c => !same(c)); next.hazards = next.hazards.filter(c => !same(c));
  if (tool === 'spawn') next.spawn = cell;
  if (tool === 'goal') { next.goal = cell; next.mode = 'escape'; }
  if (tool === 'wall') next.walls.push(cell);
  if (tool === 'gem') next.gems.push(cell);
  if (tool === 'hazard') next.hazards.push(cell);
  try { apply(next); } catch (error) { announce(error.message); }
});
sprite.addEventListener('pointerdown', event => {
  if (!ready || playing) { announce('Pause to edit the sprite'); return; }
  const [x,y] = coordinate(event, sprite,8,8), next = structuredClone(scene);
  next.sprite[y*8+x] = paint; apply(next);
});
document.querySelectorAll('[data-tool]').forEach(button => button.addEventListener('click', () => {
  tool = button.dataset.tool;
  document.querySelectorAll('[data-tool]').forEach(other => other.classList.toggle('selected', other === button));
}));
$('play').addEventListener('click', () => setPlaying(!playing));
$('reset').addEventListener('click', () => { if (ready) { apply(); announce(playing ? 'Game restarted' : 'Scene reset'); } });
$('new').addEventListener('click', () => { setPlaying(false); apply(project($('template').value)); announce('New game ready · name it, build it, then export'); });
$('preset').addEventListener('click', () => { if (ready) { setPlaying(false); apply(project()); announce('Crystal example loaded'); } });
$('grid').addEventListener('change', () => frame(false));
$('mode').addEventListener('change', () => {
  if (!ready) return;
  const next = structuredClone(scene); next.mode = $('mode').value;
  if (next.mode === 'escape' && !next.goal) {
    for (let y = 16; y > 0 && !next.goal; y--) for (let x = 30; x > 0; x--) {
      if ([next.spawn,...next.walls,...next.hazards].every(c => c[0] !== x || c[1] !== y)) { next.goal = [x,y]; break; }
    }
  }
  if (next.mode === 'collect') next.goal = null;
  setPlaying(false);
  try { apply(next); announce('Rules changed · use the Finish tool to move the goal'); } catch (error) { $('mode').value = scene.mode; announce(error.message); }
});
$('name').addEventListener('input', () => { if (scene) scene.name = $('name').value; });
$('speed').addEventListener('input', () => { if (scene) { const next = structuredClone(scene); next.speed = Number($('speed').value); apply(next); announce('Speed changed · game restarted'); } });
window.addEventListener('keydown', event => {
  if (!ready || !playing || ['INPUT','TEXTAREA','SELECT'].includes(document.activeElement.tagName)) return;
  const key = event.key.length === 1 ? event.key.toLowerCase() : event.key;
  if (['ArrowUp','ArrowDown','ArrowLeft','ArrowRight','w','a','s','d'].includes(key)) { event.preventDefault(); keys.add(key); }
});
window.addEventListener('keyup', event => keys.delete(event.key.length === 1 ? event.key.toLowerCase() : event.key));
window.addEventListener('blur', () => { keys.clear(); touch = [0,0]; });
document.querySelectorAll('[data-dir]').forEach(button => {
  button.addEventListener('pointerdown', event => { event.preventDefault(); if (!ready) return; if (!playing) setPlaying(true); touch = button.dataset.dir.split(',').map(Number); button.setPointerCapture(event.pointerId); });
  for (const event of ['pointerup','pointercancel','lostpointercapture']) button.addEventListener(event, () => touch = [0,0]);
});
function download(blob,name) {
  const url = URL.createObjectURL(blob), link = document.createElement('a'); link.href = url; link.download = name; link.click();
  setTimeout(() => URL.revokeObjectURL(url),10000);
}
$('save').addEventListener('click', () => { if (!ready) return; try { localStorage.setItem('pixelforge-demo-v1',JSON.stringify(scene)); announce('Saved in this browser'); } catch (error) { announce('Could not save: ' + error.message); } });
$('load').addEventListener('click', () => { if (!ready) return; try { const saved = localStorage.getItem('pixelforge-demo-v1'); if (!saved) throw Error('No saved game yet'); setPlaying(false); apply(JSON.parse(saved)); announce('Saved game restored'); } catch (error) { announce(error.message); } });
$('export').addEventListener('click', () => { if (ready) download(new Blob([JSON.stringify(scene,null,2)], {type:'application/json'}),'pixelforge-scene.json'); });
$('import').addEventListener('change', async event => {
  const file = event.target.files[0]; if (!file || !ready) return;
  try { if (file.size > 100000) throw Error('Scene must be smaller than 100 KB'); const next = JSON.parse(await file.text()); setPlaying(false); apply(next); announce('Game opened'); }
  catch (error) { announce('Could not open game: ' + error.message); }
  finally { event.target.value = ''; }
});
$('png').addEventListener('click', () => { if (ready) canvas.toBlob(blob => download(blob,'pixelforge-scene.png')); });
function exportedPlayer(gameScene, bytesBase64) {
  document.getElementById('title').textContent = gameScene.name;
  document.getElementById('instructions').textContent = (gameScene.mode === 'escape' ? 'Reach the green finish. Crystals are a bonus.' : 'Collect every crystal.') + ' Avoid red hazards. Arrow keys or WASD move.';
  const held = new Set(); let begun = false;
  onkeydown = event => { if (['ArrowUp','ArrowDown','ArrowLeft','ArrowRight',' '].includes(event.key)) event.preventDefault(); held.add(event.key.toLowerCase()); if (event.key.toLowerCase() === 'r' && window.pfReady) pfLoad(JSON.stringify(gameScene)); };
  onkeyup = event => held.delete(event.key.toLowerCase()); onblur = () => held.clear();
  document.getElementById('restart').onclick = () => { if (window.pfReady) { held.clear(); pfLoad(JSON.stringify(gameScene)); } };
  let direction = [0,0];
  document.querySelectorAll('[data-dir]').forEach(button => {
    button.onpointerdown = event => { event.preventDefault(); direction = button.dataset.dir.split(',').map(Number); button.setPointerCapture(event.pointerId); };
    for (const name of ['pointerup','pointercancel','lostpointercapture']) button.addEventListener(name, () => direction = [0,0]);
  });
  document.getElementById('start').onclick = async () => {
    if (begun) return; begun = true; document.getElementById('start').disabled = true;
    try {
      const go = new Go(), bytes = Uint8Array.from(atob(bytesBase64),c => c.charCodeAt(0));
      const result = await WebAssembly.instantiate(bytes,go.importObject); go.run(result.instance);
      const wait = setInterval(() => {
        if (!window.pfReady) return; clearInterval(wait); pfLoad(JSON.stringify(gameScene));
        document.getElementById('start').hidden = true;
        const context = document.querySelector('canvas').getContext('2d');
        setInterval(() => {
          const dx = direction[0] || +(held.has('arrowright') || held.has('d')) - +(held.has('arrowleft') || held.has('a'));
          const dy = direction[1] || +(held.has('arrowdown') || held.has('s')) - +(held.has('arrowup') || held.has('w'));
          context.putImageData(new ImageData(pfTick(true,dx,dy,false),320,180),0,0);
          const state = JSON.parse(pfState());
          document.getElementById('score').textContent = state.lost ? 'Game over — hit Restart to try again.' : state.won ? 'Level complete!' : state.score + '/' + state.total + ' crystals';
        },1000/30);
      },20);
    } catch (error) { document.getElementById('score').textContent = 'Could not start: ' + error.message; begun = false; document.getElementById('start').disabled = false; }
  };
}
$('html').addEventListener('click', async () => {
  if (!ready) return; const button = $('html'); button.disabled = true;
  try {
    const runtime = window.PIXELFORGE_RUNTIME_JS || await (await fetch('wasm_exec.js')).text();
    let binary = ''; const bytes = new Uint8Array(wasmBytes);
    for (let i = 0; i < bytes.length; i += 0x8000) binary += String.fromCharCode(...bytes.subarray(i,i+0x8000));
    const safeScene = JSON.stringify(scene).replace(/</g,'\\u003c');
    const html = `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>PixelForge game</title><style>body{margin:0;background:#0a1018;color:#edf2f9;text-align:center;font:16px system-ui}canvas{width:min(96vw,960px);image-rendering:pixelated;margin:24px auto;display:block}button{padding:12px 20px;cursor:pointer;background:#c7f36b;border:0;border-radius:5px;touch-action:none}h1{margin-top:32px}.controls{display:flex;justify-content:center;gap:8px;flex-wrap:wrap}</style></head><body><h1 id="title"></h1><p id="instructions"></p><button id="start">Start game</button><canvas width="320" height="180" aria-label="Your game"></canvas><p id="score"></p><div class="controls"><button id="restart">Restart / R</button><button data-dir="0,-1">↑</button><button data-dir="-1,0">←</button><button data-dir="0,1">↓</button><button data-dir="1,0">→</button></div><script>${runtime.replace(/<\/script/gi,'<\\/script')}<\/script><script>(${exportedPlayer.toString()})(${safeScene},'${btoa(binary)}');<\/script></body></html>`;
    download(new Blob([html],{type:'text/html'}),'pixelforge-playable.html'); announce('Your playable game downloaded');
  } catch (error) { announce('Export failed: ' + error.message); }
  finally { button.disabled = false; }
});
async function boot() {
  try {
    const go = new Go();
    wasmBytes = window.PIXELFORGE_WASM_BASE64 ? Uint8Array.from(atob(window.PIXELFORGE_WASM_BASE64),c => c.charCodeAt(0)).buffer : await (await fetch('engine.wasm')).arrayBuffer();
    const result = await WebAssembly.instantiate(wasmBytes,go.importObject); go.run(result.instance).catch(error => announce('Engine stopped: ' + error.message));
    const poll = setInterval(() => {
      if (!window.pfReady) return; clearInterval(poll); ready = true; palette = JSON.parse(window.pfPalette());
      palette.slice(0,16).forEach((color,i) => {
        const button = document.createElement('button'); button.style.background = color; button.title = 'Colour ' + i; button.setAttribute('aria-label','Sprite colour ' + i); button.classList.toggle('selected',i === paint);
        button.onclick = () => { paint = i; document.querySelectorAll('#swatches button').forEach(other => other.classList.toggle('selected',other === button)); }; $('swatches').append(button);
      });
      $('loading').style.display = 'none'; $('engineDot').style.background = '#c7f36b';
      for (const id of ['play','reset','new']) $(id).disabled = false;
      apply(window.PIXELFORGE_SCENE || project()); announce('Go engine ready · start a project or edit this example'); requestAnimationFrame(loop);
    },20);
  } catch (error) { $('loading').textContent = 'Engine could not load. ' + error.message; announce('Load failed'); }
}
boot();
