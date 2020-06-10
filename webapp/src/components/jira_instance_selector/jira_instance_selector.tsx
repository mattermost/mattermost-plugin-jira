import React from 'react';

import {Instance, ReactSelectOption, GetConnectedResponse} from 'types/model';

import ReactSelectSetting from 'components/react_select_setting';

export type Props = {
    instances: Instance[];
    connectedInstances: Instance[];
    value: string;
    theme: {};
    getConnected: () => Promise<GetConnectedResponse>;
    onChange: (instanceID: string) => void;
    onlyShowConnectedInstances?: boolean;
}

export type State = {
    error?: string;
}

export default class JiraInstanceSelector extends React.PureComponent<Props, State> {
    state = {
        error: '',
    };

    componentDidMount() {
        this.fetchInstances();
    }

    fetchInstances = async () => {
        const {data, error} = await this.props.getConnected();
        if (error) {
            this.setState({error: error.toString()});
        }
    }

    private onChange = (_: string, instanceID: string) => {
        this.props.onChange(instanceID);
    }

    public render(): JSX.Element {
        let error;
        if (this.state.error) {
            error = (
                <span>{this.state.error}</span>
            );
        }

        let instances = this.props.instances;
        if (this.props.onlyShowConnectedInstances) {
            instances = this.props.connectedInstances;
        }

        const options: ReactSelectOption[] = instances.map((instance: Instance) => (
            {label: instance.instance_id, value: instance.instance_id}
        ));

        return (
            <React.Fragment>
                <ReactSelectSetting
                    options={options}
                    theme={this.props.theme}
                    onChange={this.onChange}
                    value={options.find((opt) => opt.value === this.props.value)}
                />
                {error}
            </React.Fragment>
        );
    }
}
