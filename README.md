# imageresizeserver

## Description

Supports resizing .jpeg (.jpg) .png and .wepb images on the fly.

# Env
```
GIN_MODE=debug

CORS_ALLOW_ORIGINS="*"

LOCAL_FS_ROOT_DIRECTORY=/data
```

# Run

## Go
```
go run main.go
```

## Docker
```
docker-compose -p imageresizeserver up -d
```

# Usage

Upload an image to the server and get the resized image back.

Original image server location: /data/path/to/test_image.jpg
Original image URL: http://localhost:8080/path/to/test_image.jpg

Resized image server location: /data/path/to/1080xAUTO/test_image.jpg
Resized image URL: http://localhost:8080/path/to/1080xAUTO/test_image.jpg

# Django serializer integration example

```python
import os

from django.conf import settings
from rest_framework import serializers


class HyperlinkedSorlImageField(serializers.ImageField):
    def __init__(self, geometry=None, options={}, *args, **kwargs):
        """
        Create an instance of the HyperlinkedSorlImageField image serializer.
        Args:
            geometry_string (str): The size of your cropped image.
            options (Optional[dict]): A dict of sorl options.
            *args: (Optional) Default serializers.ImageField arguments.
            **kwargs: (Optional) Default serializers.ImageField keyword
            arguments.
        For a description of sorl geometry strings and additional sorl options,
        please see https://sorl-thumbnail.readthedocs.org/en/latest/examples.html?highlight=geometry#low-level-api-examples
        """  # NOQA
        self.geometry = geometry
        self.options = options

        super(HyperlinkedSorlImageField, self).__init__(**kwargs)

    def to_representation(self, value):
        """
        Perform the actual serialization.

        Args:
            value: the image to transform
        Returns:
            a url pointing at a scaled and cached image
        """
        if not value:
            return None

        url = settings.RESIZER_MEDIA_URL  # localhost:8080
        head, tail = os.path.split(str(value))

        if not self.geometry:
            return f"{url}/{head}/{tail}"

        geometry = str(self.geometry)
        if "x" not in geometry:
            geometry = self.geometry + "xAUTO/"
        else:
            geometry += "/"
        return url + "/" + head + "/" + geometry + tail
```