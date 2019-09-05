import React from 'react';

import {FilterField, FilterValue} from 'types/model';

import ChannelSettingsFilter from './channel_settings_filter';

type ChannelSettingsFiltersProps = {
    fields: FilterField[];
    values: FilterValue[];
    theme: object;
    addValidate: () => void;
    removeValidate: () => void;
    onChange: (f: FilterValue[]) => void;
};

type ChannelSettingsFiltersState = {
    showCreateRow: boolean;
};

export default class ChannelSettingsFilters extends React.PureComponent<ChannelSettingsFiltersProps, ChannelSettingsFiltersState> {
    state = {
        showCreateRow: false,
    };

    onConfiguredValueChange = (oldValue: FilterValue | null, newValue: FilterValue) => {
        const newValues = this.props.values.concat([]);
        const index = newValues.findIndex((f) => f === oldValue);

        if (index === -1) {
            newValues.push({exclude: false, values: [], ...newValue});
            this.setState({showCreateRow: false});
        } else {
            newValues.splice(index, 1, newValue);
        }

        this.props.onChange(newValues);
    };

    addNewFilter = () => {
        this.setState({showCreateRow: true});
    };

    hideNewFilter = () => {
        this.setState({showCreateRow: false});
    };

    removeFilter = (value: FilterValue | null) => {
        if (!value) {
            return;
        }

        const newValues = this.props.values.concat([]);
        const index = newValues.findIndex((f) => f === value);

        if (index !== -1) {
            newValues.splice(index, 1);
        }
        this.props.onChange(newValues);
    };

    render() {
        const {fields, values} = this.props;
        const {showCreateRow} = this.state;

        return (
            <ul style={{listStyleType: 'none'}}>
                {values.map((v, i) => {
                    const field = fields.find((f) => f.key === v.key);
                    if (!field) {
                        return null;
                    }
                    return (
                        <li key={i}>
                            <ChannelSettingsFilter
                                fields={fields}
                                field={field}
                                value={v}
                                onChange={this.onConfiguredValueChange}
                                removeFilter={this.removeFilter}
                                theme={this.props.theme}
                                addValidate={this.props.addValidate}
                                removeValidate={this.props.removeValidate}
                            />
                        </li>
                    );
                })}
                {showCreateRow && (
                    <li>
                        <ChannelSettingsFilter
                            fields={fields}
                            field={null}
                            value={null}
                            onChange={this.onConfiguredValueChange}
                            theme={this.props.theme}
                            removeFilter={this.hideNewFilter}
                        />
                    </li>
                )}
                <button
                    onClick={this.addNewFilter}
                    disabled={showCreateRow}
                    className='btn btn-info'
                >
                    {'Add Filter'}
                </button>
            </ul>
        );
    }
}
