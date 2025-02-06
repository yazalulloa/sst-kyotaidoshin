DIR=$(dirname "$(readlink -f "$0")")
RES_DIR=$DIR/src/assets


if [ -n "$1" ]
then
  echo "Minify"
  bun build --minify --target=browser --outdir="$RES_DIR" "$DIR"/js
else
  echo "Watching"
  bun build --target=browser --outdir="$RES_DIR" "$DIR"/js --watch
fi


#bun build --minify  --outdir="$RES_DIR"/out "$RES_DIR"/js  --watch
#bun build --outdir="$RES_DIR"/out "$RES_DIR"/js  --watch

