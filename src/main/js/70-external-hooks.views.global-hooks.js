var ViewGlobalHooks = function (context, api) {
    this._$ = $('#rq_hooks_global_hooks_form');
    if (this._$.length == 0) {
        return new ViewNotApplicable();
    }

    this._triggers = new ViewGlobalSettings(context, api)

    this._$spinner = new Spinner();

    this.mount = function () {
        this._$.find('h2').append(this._$spinner);

        this._$.find('[data-rq-hook-kind]').each(function () {
            var $pane = $(this);
            var kind = $pane.attr('data-rq-hook-kind');

            $pane.find('[name$=".enabled"]').change(function () {
                $pane.find('.rq-global-hook-settings').
                    toggleClass('rq-hook-disabled', !$(this).prop('checked'));
            })
        });

        this._$.submit(function (e) {
            e.preventDefault();

            this._withLoader(this._saveSettings.bind(this));
        }.bind(this));

        this._withLoader(this._loadSettings.bind(this));
    }

    this._withLoader = function (fn) {
        this._setLoading(true);
        fn().then(this._setLoading.bind(this, false));
    }

    this._setLoading = function (loading) {
        if (loading) {
            this._$.find('input, button, textarea').prop('disabled', true);
            this._$spinner.show();
        } else {
            this._$.find('input, button, textarea').prop('disabled', false);
            this._$spinner.hide();
        }
    }

    this._saveSettings = function () {
        var updating = this._updateSettings();

        var flags = {
            ok: new FlagSuccess('Settings successfully updated.'),
            error: new FlagError('Settings were not applied due to validation error.')
        }

        this._triggers._$progress.remove();

        updating = updating.then(function () {
            var error = false;

            $.each(arguments, function (_, data) {
                var tab = this._getTabByKind(data.kind);

                if ('errors_fields' in data.response) {
                    error = true;
                    $.each(data.response.errors_fields, function(field, errors) {
                        tab.pane.find('#hooks-' + tab.id + '-' + field + '-error').text(
                            errors.join(' ')
                        );
                    }.bind(this));
                    tab.tab
                        .removeClass('rq-tab-ok')
                        .addClass('rq-tab-error');
                } else {
                    tab.pane.find('.rq-hook-field-error').text('');
                    tab.tab
                        .removeClass('rq-tab-error')
                        .toggleClass('rq-tab-ok', data.settings.enabled);
                }
            }.bind(this));

            if (error) {
                flags.error.show();
            } else {
                if (this._$.find('[name="apply-existing"]').prop('checked')) {
                  return this._triggers._applySettings()
                        .then(this._triggers._renderApplyProgress.bind(this._triggers))
                        .then(this._triggers._monitorApplyProgress.bind(this._triggers))
                        .then(flags.ok.show.bind(flags.ok));
                }
            }
        }.bind(this));

        return updating;
    }

    this._updateSettings = function () {
        var settings = this._getSettings();

        return $.when.apply(
            $,
            $.map(settings.hooks, function (settings, kind) {
                return api.setGlobalHook(kind, settings)
                    .then(function(response) {
                        return {
                            kind: kind,
                            settings: settings,
                            response: response,
                        }
                    });
            })
        );
    }

    this._getTabByKind = function(kind) {
        var $pane = this._$.find('[data-rq-hook-kind="' + kind + '"]')
        var id = $pane.attr('id').replace(/^hooks-/, '')
        var $tab = this._$.find('#hooks-' + id + '-tab')

        return {
            tab: $tab,
            pane: $pane,
            id: id,
        };
    }

    this._applySettings = function () {
        return api.runHooksFactory();
    }

    this._renderSettings = function (kind, settings) {
        var tab = this._getTabByKind(kind);
        tab.tab.toggleClass('rq-tab-ok', settings.enabled);

        $.each(settings, function (k, v) {
            var $input = tab.pane.find('[name="hooks.' + tab.id + '.' + k + '"]');
            if ($input.is('[type="checkbox"]')) {
                $input.attr('checked', v).trigger('change');
            } else if ($input.is('[type="radio"]')) {
              $input.filter('[value="' + v + '"]').
                attr('checked', v).
                trigger('change');
            } else {
                $input.val(v);
            }
        });
    }

    this._loadSettings = function () {
        var kinds = this._$.find('[data-rq-hook-kind]').
            map(function (_, el) { return el.getAttribute('data-rq-hook-kind'); }).
            get();

        return $.when.apply(
            $,
            kinds.map(function(kind) {
                return api.getGlobalHook(kind).then(function (settings) {
                    return {
                        kind: kind,
                        settings: settings
                    };
                })
            })
        ).done(function() {
            $.each(arguments, function (_, hook) {
                this._renderSettings(hook.kind, hook.settings);
            }.bind(this))
        }.bind(this));

        //return api.getSettings()
        //    .done(this._renderSettings.bind(this));
    }

    this._loadSettingsDefaults = function () {
        return api.getSettingsDefaults()
            .done(this._renderSettings.bind(this));
    }

    this._getSettings = function() {
        var hooks = {};

        this._$.find('[name^="hooks."]')
            .each(
                function () {
                    var matches = $(this).attr('name')
                        .match(/hooks\.(\S+)\.(\S+)/);

                    var kind = $(this).parents('[data-rq-hook-kind]').
                        attr('data-rq-hook-kind')

                    if (!hooks[kind]) {
                        hooks[kind] = {}
                    }

                  var value;

                  if ($(this).is('[type="checkbox"]')) {
                    value = !!$(this).attr('checked');
                  } else if ($(this).is('[type="radio"]')) {
                    value = $(this).filter(':checked').val();
                  } else {
                    value = $(this).val();
                  }

                  if (value !== undefined) {
                    hooks[kind][matches[2]] = value;
                  }
                }
            );

        return {
            'hooks': hooks
        }
    }

    return this;
}
