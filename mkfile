<$PLAN9/src/mkhdr

BUGGERED='nada'
DIRS=`ls -l |sed -n 's/^d.* //p' |egrep -v "^($BUGGERED)$"|egrep -v '^lex$'`

install:V:
	for i in $DIRS
	do
		(cd $i; echo cd `pwd`';' go build -o $PLAN9/bin/$i .; go build -o $PLAN9/bin/$i .)
	done
