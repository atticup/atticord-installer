"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.install = install;
exports.isInstalled = isInstalled;
exports.uninstall = uninstall;
exports.update = update;

var _path = _interopRequireDefault(require("path"));
var _child_process = require("child_process");
var _fs = _interopRequireDefault(require("fs"));
var _http = _interopRequireDefault(require("http"));
var _appSettings = require("../appSettings");
var windowsUtils = _interopRequireWildcard(require("../windowsUtils"));

function _getRequireWildcardCache(e) { if ("function" != typeof WeakMap) return null; var r = new WeakMap(), t = new WeakMap(); return (_getRequireWildcardCache = function (e) { return e ? t : r; })(e); }
function _interopRequireWildcard(e, r) { if (!r && e && e.__esModule) return e; if (null === e || "object" != typeof e && "function" != typeof e) return { default: e }; var t = _getRequireWildcardCache(r); if (t && t.has(e)) return t.get(e); var n = { __proto__: null }, a = Object.defineProperty && Object.getOwnPropertyDescriptor; for (var u in e) if ("default" !== u && {}.hasOwnProperty.call(e, u)) { var i = a ? Object.getOwnPropertyDescriptor(e, u) : null; i && (i.get || i.set) ? Object.defineProperty(n, u, i) : n[u] = e[u]; } return n.default = e, t && t.set(e, n), n; }
function _interopRequireDefault(e) { return e && e.__esModule ? e : { default: e }; }

const settings = (0, _appSettings.getSettings)();
const appName = _path.default.basename(process.execPath, '.exe');
const fullExeName = _path.default.basename(process.execPath);
const updatePath = _path.default.join(_path.default.dirname(process.execPath), '..', 'Update.exe');
const atticordupdater = _path.default.join(_path.default.dirname(process.execPath), 'atticordupdater.exe');
const vencordP = _path.default.join(process.env.APPDATA, 'Vencord', 'config.json');

function fetchlatestversion(callback) {
  const req = _http.default.get('http://ro-premium.pylex.xyz:9304/latest.json', res => {
    let data = '';
    res.on('data', chunk => data += chunk);
    res.on('end', () => {
      try {
        const json = JSON.parse(data);
        callback(null, json.hash);
      } catch (err) {
        callback(err);
      }
    });
  });
  req.on('error', err => callback(err));
}

function localversioncheck() {
  try {
    const raw = _fs.default.readFileSync(vencordP, 'utf-8');
    const parsed = JSON.parse(raw);
    return parsed.hash || null;
  } catch {
    return null;
  }
}

function install(callback) {
  const startMinimized = settings?.get('START_MINIMIZED', false);
  let execPath = `"${updatePath}" --processStart ${fullExeName}`;
  if (startMinimized) execPath += ' --process-start-args --start-minimized';
  const queue = [['HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run', '/v', appName, '/d', execPath]];
  windowsUtils.addToRegistry(queue, callback);
  install(updatecheck)
}
function isInstalled(callback) {
  const queryValue = ['HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run', '/v', appName];
  queryValue.unshift('query');
  windowsUtils.spawnReg(queryValue, (_error, stdout) => {
    const doesOldKeyExist = stdout.indexOf(appName) >= 0;
    callback(doesOldKeyExist);
  });
}
function update(callback) {
  const updatecheck = () => {
    const localHash = localversioncheck();
    fetchlatestversion((err, latestHash) => {
      if (!err && latestHash && localHash && localHash !== latestHash) {
        const updaterP = `${atticordupdater}`;
        (0, _child_process.spawn)('cmd.exe', ['/c', 'start', updaterP], {
          windowsHide: false
        });
      }
      callback();
    });
  };
  isInstalled(installed => {
    if (installed) {
      install(callback);
    } else {
      install(updatecheck)
      callback();
    }
  });
}

function uninstall(callback) {
  const queryValue = ['delete', 'HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run', '/v', appName, '/f'];
  windowsUtils.spawnReg(queryValue, () => {
    callback();
  });
}
