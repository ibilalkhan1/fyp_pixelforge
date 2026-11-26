/* PixelForge Pages host: same supplied Emscripten runtime, with local demo
   cartridges installed after IndexedDB restores and before Studio starts. */
var Module = window.Module = {
  canvas: document.getElementById('canvas'),
  arguments: [],
  locateFile: function (name) { return new URL(name, document.baseURI).href; },
  printErr: function (message) { console.error('[PixelForge]', message); },
  onAbort: function () { showFailure('Studio could not start. Reload the page to retry.'); },
  onRuntimeInitialized: function () {
    var start = Module._sokol_start;
    Module._sokol_start = function () {
      var directory = '/com.pixelforge/PixelForge';
      FS.mkdirTree(directory);
      installedCartridges.forEach(function (cart) {
        var path = directory + '/' + cart.name;
        // Respect cartridges visitors have already edited and saved.
        if (!FS.analyzePath(path).exists) FS.writeFile(path, cart.bytes);
      });
      return start.apply(this, arguments);
    };
  }
};
var installedCartridges = [];
var launchButton = document.getElementById('game-frame');
var launchStarted = false;
function showFailure(message) {
  launchButton.hidden = false;
  launchButton.disabled = false;
  launchButton.textContent = message;
  launchButton.onclick = function () { location.reload(); };
}
// The runtime asks for /export and /js resources from the domain root.
// GitHub project Pages lives below /fyp_pixelforge/, so keep those requests
// within this site's directory. Do not rewrite unrelated requests.
var originalOpen = XMLHttpRequest.prototype.open;
XMLHttpRequest.prototype.open = function (method, url) {
  var args = Array.prototype.slice.call(arguments);
  if (typeof url === 'string' && /^\/(export|js)\//.test(url)) {
    args[1] = new URL(url.slice(1), document.baseURI).href;
  }
  return originalOpen.apply(this, args);
};
launchButton.addEventListener('click', async function () {
  if (launchStarted) return;
  launchStarted = true;
  launchButton.disabled = true;
  launchButton.textContent = 'Loading Studio and cartridges…';
  try {
    var manifestResponse = await fetch('cartridges/manifest.json');
    if (!manifestResponse.ok) throw new Error('Cartridge list unavailable');
    var manifest = await manifestResponse.json();
    installedCartridges = await Promise.all(manifest.map(async function (cart) {
      if (!/^[a-zA-Z0-9_-]+\.pf$/.test(cart.name)) throw new Error('Invalid cartridge name');
      var response = await fetch(cart.url);
      if (!response.ok) throw new Error('Cartridge unavailable: ' + cart.name);
      return { name: cart.name, bytes: new Uint8Array(await response.arrayBuffer()) };
    }));
    // Hide only after downloads succeed. The click is also the browser gesture
    // used by the supplied runtime's audio/input initialization.
    launchButton.hidden = true;
    var script = document.createElement('script');
    script.src = 'pixelforge.js?v=20261126';
    script.onerror = function () { showFailure('Runtime download failed. Click to reload.'); };
    document.body.appendChild(script);
    Module.canvas.focus();
  } catch (error) {
    console.error(error);
    showFailure('Download failed. Check your connection, then click to reload.');
  }
});
