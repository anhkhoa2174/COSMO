(cd out/ &&
  find . -type f -name '*.html' | while read HTMLFILE; do
    HTMLFILESHORT=${HTMLFILE:2}
    HTMLFILE_WITHOUT_INDEX=${HTMLFILESHORT::${#HTMLFILESHORT}-11}

    echo $HTMLFILESHORT
    echo $HTMLFILE_WITHOUT_INDEX
    # cp /about/index.html to /about
    # aws s3 cp s3://$S3_BUCKET/${HTMLFILESHORT} \
    #   s3://$S3_BUCKET/$HTMLFILE_WITHOUT_INDEX

    if [ $? -ne 0 ]; then
      echo "***** Failed renaming build to $S3_BUCKET/$NAMESPACE (html)"
      exit 1
    fi
  done)
