# Fonts

`nohemi-variable.woff2` is deliberately not committed. Nohemi is a licensed
typeface, and this repository is public, so publishing the binary here would be
redistribution.

To run the app locally, convert the variable font once:

```sh
npm i -D wawoff2
node -e "const w=require('wawoff2'),f=require('fs');\
w.compress(f.readFileSync('Nohemi-VF.ttf')).then(o=>f.writeFileSync('static/fonts/nohemi-variable.woff2',o))"
npm rm wawoff2
```

Without it the interface falls back to the system sans stack and still renders
correctly, including Arabic and Bengali.
