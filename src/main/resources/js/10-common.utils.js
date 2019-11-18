//
// Utility classes.
//

var Options = function (options, defaults) {
    return $.extend(defaults, options);
}

var Query = function (data) {
    var components = [];

    $.each(data, function(key, value) {
        if (value !== null) {
            components.push(key + "=" + encodeURIComponent(value));
        }
    });

    return components.join("&");
}

var React_16 = function (element) {
    var element = $(element)[0];
    if (!element._reactRootContainer) {
        return null;
    }

    this.state = function() {
        var child = element.
            _reactRootContainer.
                _internalRoot.
                    current.
                        child;

        // BB 5.12.0 has it's state one level deeper.
        return (child.stateNode || child.child.stateNode).state;
    }

    return this;
}

var React_15 = function (element) {
    var element = $(element)[0];
    var key = Object.keys(element).find(function (key) {
        return key.startsWith("__reactInternalInstance$");
    });

    if (!key) {
        this.state = function() {
            return null;
        }

        return this;
    } else {
        var pointer = element[key];
        while (pointer._currentElement._owner != null) {
            pointer = pointer._currentElement._owner;
        }

        this.state = function() {
            return pointer._instance.state;
        }

        return this;
    }
}

var Observer = function (selector, fn) {
    var MutationObserver =
        window.MutationObserver ||
        window.WebKitMutationObserver;

    this._observer = new MutationObserver(
        function(mutations, observer) {
            var timeout = null;

            $.each(mutations, function (index, mutation) {
                var $target = $(mutation.target);

                if ($target.filter(selector).length > 0) {
                    if (timeout != null) {
                        clearTimeout(timeout);
                    }

                    timeout = setTimeout(fn($target), 10)
                }
            });
        }
    );

    this.observe = function (target) {
        this._observer.observe(
            $(target)[0],
            {subtree: true, childList: true}
        );
    }

    return this;
}

var ViewNotApplicable = function () {
    this.mount = function () {
        return null;
    }

    return this;
}

var Colors = {
    FromHex: function (hex) {
        if (!hex.match(/^#[\da-f]{6}$/i)) {
            return null;
        }

        var color = parseInt(hex.slice(1), 16);

        return {
            r: 0xFF & (color >> 16),
            g: 0xFF & (color >> 8),
            b: 0xFF & color
        };
    },

    ToHex: function (rgb) {
        return "#" +
            ((1 << 24) + (rgb.r << 16) + (rgb.g << 8) + rgb.b)
                .toString(16).slice(1);
    },

    Luminance: function (rgb) {
        // https://stackoverflow.com/a/24213274
        return Math.sqrt(
            0.299 * rgb.r*rgb.r + 0.587 * rgb.g * rgb.g + 0.114 * rgb.b * rgb.b
        );
    },

    IsBright: function(rgb) {
        return Colors.Luminance(rgb) > 186;
    }
}

