// Converts a flat DB row (snake_case columns) into a camelCase object for
// JSON responses, so the API speaks one consistent casing regardless of the
// SQL convention underneath.
function toCamelCase(row) {
  if (!row) return row;
  const result = {};
  for (const [key, value] of Object.entries(row)) {
    const camelKey = key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
    result[camelKey] = value;
  }
  return result;
}

module.exports = { toCamelCase };
