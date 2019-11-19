//
// UI elements library.
//
// Components that can be reused in different project defined there.
// These components can be insterted into DOM hierarchy directly since they
// inherit jQuery object.
//

var Icon = function(icon, options) {
    var options = new Options(options, {
        size: 'small',
        classes: []
    });

    return $(aui.icons.icon({
        useIconFont: true,
        size: options.size,
        icon: icon,
        extraClasses: options.classes,
    }));
}

var ButtonIcon = function(icon, options) {
    var options = Options(options, {
        classes: []
    });

    return $(aui.buttons.button({
        text: '',
        type: 'subtle',
        iconType: 'aui',
        iconClass: 'aui-icon-small aui-iconfont-' + icon,
        extraClasses: options.classes
    }));
}

var Label = function(label, options) {
    var options = Options(options, {
        on: {}
    });

    options.on = Options(options.on, {
        click: function () {},
        close: null
    })

    var config = {
        text: label.name,
        isCloseable: $.isFunction(options.on.close),
        extraClasses: ['rq-label']
    }

    var $node;

    if (aui.labels) {
        $node = $(aui.labels.label(config));
    } else {
        $node = $('<span class="aui-label"/>').text(label.name);

        if ($.isFunction(options.on.close)) {
            this._$.
                addClass("aui-label-closeable").
                append($('<span class="aui-icon aui-icon-close"/>'))
        }
    }

    $node.
        find('.aui-icon-close').
        click(function(e) {
            e.stopPropagation();
            options.on.close.bind($(this).parent())();
        }).
        end();

    this.color = function (bg) {
        if (bg == null) {
            return this.uncolor();
        }

        var fg = Colors.IsBright(Colors.FromHex(bg))
            ? '#000000'
            : '#FFFFFF';

        $node
            .css('background-color', bg)
            .css('color', fg)
            .css('border-color', bg);
    }

    this.uncolor = function () {
        $node
            .css('background-color','')
            .css('color','')
            .css('border-color','');
    }

    this._$ = $.extend($node, this);

    $node.click(options.on.click.bind(this._$));

    if (label.color) {
        this.color(label.color);
    }

    return this._$;
}

var Spinner = function(options) {
    var options = Options(options, {
        size: 'small'
    });

    return $('<aui-spinner class="rq-spinner"/>').
        attr("size", options.size).
        hide();
}

var Message = function (severity, content, options) {
    var options = Options(options, {
        fade: Options(options && options.fade || {}, {
            delay: 2000,
            duration: 2000
        })
    });

    this._$ = $(aui.message[severity]({
        content: content || ""
    })).hide();

    return $.extend(this._$, {
        fadeout: function () {
            this.
                delay(options.fade.delay).
                fadeOut(options.fade.duration);
            return this;
        }
    });
}

var MessageError = function (content) {
    return new Message('error', content);
}

var Select = function (options) {
    var __noop__ = function() {}

    var options = Options(options, {
        allowNew: false,
        placeholder: '...',
        query: function(_) { return [] },
        itemize: function(_) { return term },
        empty: '<empty>',
        css: Options(options.css, {
            container: '',
            dropdown: ''
        }),
        on: Options(options.on, {
            create: __noop__,
            clear: __noop__
        }),
    });

    var config = {
        placeholder: options.placeholder,
        allowClear: true,
        query: function (args) {
            var data = [];

            var term = args.term.trim();

            var source = options.query();
            var hit = null;
            var matches = $.grep(source, function (item) {
                if (item.name == term) {
                    hit = item;
                }

                return item.name.startsWith(term)
            });

            // push hit first
            if (hit) {
                data.push(hit);
            }

            $.each(matches, function(_, item) {
                if (hit && hit.id == item.id || item.disabled) {
                    return;
                }

                data.push(item);
            });

            if (options.allowNew) {
                if (!hit && term != "") {
                    data.push({
                        id: '',
                        name: term,
                    });
                }
            }

            return args.callback({
                results: data
            });
        },
        formatResult: options.itemize,
        formatSelection: options.itemize,
        formatNoMatches: function() {
            return options.empty;
        },
        dropdownCssClass: options.css.dropdown,
        containerCssClass: options.css.container
    }

    if ($.isFunction(options.on.create) && options.on.create != __noop__) {
        config.formatSelection = function(item) {
            item.name = item.name.trim();
            return options.itemize(options.on.create(item))
        }

        config.createSearchChoice = function (term, data) {
            var term = term.trim();

            if (data.length > 0) {
                return null;
            }

            return {
                id: '',
                name: term,
            };
        }
    }

    this._$ = $('<span><input/>').
        children().
            auiSelect2(config).
                // We need to overload click event to be able to clear
                // underlying Select object, because 'allowClear'
                // configuration option for Select2 does not work here for
                // whatever reason.
                on('select2-opening', function(e) {
                    var value = $(this).select2('data');

                    if (value) {
                        var event = $.Event('change');
                        event.removed = value;

                        $(this).select2('val', '');
                        $(this).trigger(event);
                        options.on.clear();
                        e.preventDefault();
                    }
                }).
        end();

    return $.extend(this._$, {
        disable: function() {
            this.children().select2('readonly', true);
        },

        enable: function() {
            this.children().select2('readonly', false);
        },

        empty: function() {
            this.children().select2('val', '');
        },

        change: function(fn) {
            this.children().on('change', fn);
        },

        close: function() {
            this.children().select2('close');
        },

        itemize: function(item) {
            return options.itemize(item);
        }
    });
}

var Popup = function (id, body, options) {
    var options = Options(options, {
        alignment: "left top"
    });

    this._$template = $(new AJS.InlineDialog2())
        .attr('id', id)
        .attr('alignment', options.alignment);

    this._$anchor = null

    return $.extend(this._$template, {
        close: function () {
            if (this._$) {
                this._$.removeAttr('open');
            }
        }.bind(this),

        open: function (anchor) {
            if (this._$) {
                this._$.detach();
            }

            if (this._$anchor != null) {
                this._$anchor.removeAttr('aria-controls');
            }

            this._$ = this._$template
                .clone()
                .find('div')
                    .append(body)
                .end()
                .appendTo('body');

            this._$anchor = $(anchor);

            this._$anchor.attr('aria-controls', id);

            this._$.attr('open', true);
        }.bind(this)
    });
}

var Nagbar = function (body) {
    this._shown = false;
    this._body = body;

    this.show = function () {
        if (this._shown) {
            return;
        }

        require('aui/banner')({ body: this._body });
        this._shown = true;
    }

    return this;
}

var ProgressBar = function () {
    this._$ = $('<aui-progressbar/>')

    return $.extend(this._$, {
        setIndeterminate: function (value) {
            if (value) {
                return this.attr('indeterminate', 'indeterminate');
            } else {
                return this.removeAttr('indeterminate');
            }
        },
        setCurrent: function (value) {
            return this.attr('value', value)
        },
        setTotal: function (value) {
            return this.attr('max', value)
        }
    });
}

var ProgressBarWithText = function () {
    this._$text = $('<span class="rq-progress-text">');
    this._$bar = new ProgressBar();

    this._$ = $('<div class="rq-progress"/>')
        .append(this._$bar)
        .append(this._$text);

    return $.extend(this._$, {
        setIndeterminate: function (value) {
            this._$bar.setIndeterminate(value)
            return this._$;
        }.bind(this),

        setCurrent: function (value) {
            this._$bar.setCurrent(value)
            return this._$;
        }.bind(this),

        setTotal: function (value) {
            this._$bar.setTotal(value)
            return this._$;
        }.bind(this),

        setText: function (value) {
            this._$text.text(value)
            return this._$;
        }.bind(this)
    });
}
