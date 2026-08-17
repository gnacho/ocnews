module.exports = {
  input: {
    path: './src',
    include: ['**/*.js', '**/*.ts', '**/*.vue']
  },
  output: {
    locales: ['es', 'en'],
    path: './l10n/locale',
    potPath: '../template.pot',
    jsonPath: '../translations.json',
    flat: false,
    linguas: false
  }
}
