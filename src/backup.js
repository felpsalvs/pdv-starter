const fs = require('fs');
const path = require('path');
const db = require('./db');

const BACKUP_DIR = path.join(__dirname, '..', 'backups');
const INTERVAL_MS = 4 * 60 * 60 * 1000; // 4 horas
const RETENTION_COUNT = 30;

function pad(n) {
  return String(n).padStart(2, '0');
}

function backupFileName() {
  const d = new Date();
  const date = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
  const time = `${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`;
  return `pdv-${date}-${time}.db`;
}

function pruneOldBackups() {
  let files;
  try {
    files = fs
      .readdirSync(BACKUP_DIR)
      .filter((name) => name.startsWith('pdv-') && name.endsWith('.db'))
      .sort();
  } catch (err) {
    return;
  }

  const excess = files.length - RETENTION_COUNT;
  if (excess <= 0) return;

  for (const name of files.slice(0, excess)) {
    try {
      fs.unlinkSync(path.join(BACKUP_DIR, name));
    } catch (err) {
      // Se não conseguir apagar um backup antigo, não é motivo pra travar nada.
    }
  }
}

// Usa o backup nativo do SQLite (via better-sqlite3), seguro mesmo com o
// banco aberto em modo WAL — diferente de copiar o arquivo .db na mão, que
// pode pegar uma cópia inconsistente enquanto o sistema está rodando.
async function createBackup() {
  try {
    fs.mkdirSync(BACKUP_DIR, { recursive: true });
    const destination = path.join(BACKUP_DIR, backupFileName());
    await db.backup(destination);
    pruneOldBackups();
    console.log(`Backup do banco salvo em backups/${path.basename(destination)}`);
  } catch (err) {
    console.error('Não foi possível fazer o backup automático do banco:', err.message);
  }
}

function scheduleBackups() {
  createBackup();
  const interval = setInterval(createBackup, INTERVAL_MS);
  interval.unref();

  const backupAndExit = () => {
    createBackup().finally(() => process.exit(0));
  };
  process.on('SIGINT', backupAndExit);
  process.on('SIGTERM', backupAndExit);
}

module.exports = { scheduleBackups, createBackup };
