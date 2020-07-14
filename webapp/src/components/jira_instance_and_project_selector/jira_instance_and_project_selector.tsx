import React from 'react';

import {Theme} from 'mattermost-redux/types/preferences';

import {Instance, ProjectMetadata, ReactSelectOption, APIResponse, GetConnectedResponse} from 'types/model';
import ReactSelectSetting from 'components/react_select_setting';
import {getProjectValues} from 'utils/jira_issue_metadata';

export type Props = {
    selectedInstanceID: string | null;
    selectedProjectID: string | null;
    onInstanceChange: (instanceID: string) => void;
    onProjectChange: (projectID: string) => void;
    onError: (err: string) => void;

    theme: Theme;
    addValidate: (isValid: () => boolean) => void;
    removeValidate: (isValid: () => boolean) => void;

    installedInstances: Instance[];
    connectedInstances: Instance[];
    defaultUserInstanceID?: string;
    fetchJiraProjectMetadata: (instanceID: string) => Promise<APIResponse<ProjectMetadata>>;
    getConnected: () => Promise<GetConnectedResponse>;
    hideProjectSelector?: boolean;
};

type State = {
    fetchingInstances: boolean;
    fetchingProjectMetadata: boolean;
    jiraProjectMetadata: ProjectMetadata | null;
    disableInstanceSelector: boolean;
};

export default class JiraInstanceAndProjectSelector extends React.PureComponent<Props, State> {
    constructor(props: Props) {
        super(props);

        this.state = {
            fetchingProjectMetadata: false,
            fetchingInstances: false,
            jiraProjectMetadata: null,
            disableInstanceSelector: Boolean(this.props.selectedInstanceID),
        };
    }

    componentDidMount() {
        this.fetchInstances();
    }

    fetchInstances = async () => {
        if (this.props.selectedInstanceID) {
            this.props.onInstanceChange(this.props.selectedInstanceID);
            if (!this.props.hideProjectSelector) {
                this.fetchJiraProjectMetadata(this.props.selectedInstanceID);
            }
            return;
        }

        this.setState({fetchingInstances: true});
        const {error} = await this.props.getConnected();
        this.setState({fetchingInstances: false});
        if (error) {
            this.props.onError(error.message);
            return;
        }

        let instanceID = '';
        if (this.props.connectedInstances.length === 1) {
            instanceID = this.props.connectedInstances[0].instance_id;
        } else if (this.props.defaultUserInstanceID) {
            instanceID = this.props.defaultUserInstanceID;
        }

        if (instanceID) {
            this.handleJiraInstanceChange('', instanceID);
        }
    }

    fetchJiraProjectMetadata = async (instanceID: string) => {
        if (!this.state.fetchingProjectMetadata) {
            this.setState({jiraProjectMetadata: null, fetchingProjectMetadata: true});
        }

        const {data, error} = await this.props.fetchJiraProjectMetadata(instanceID);
        if (error) {
            this.setState({fetchingProjectMetadata: false});
            this.props.onError(error.message);
            return;
        }

        const projectMetadata = data as ProjectMetadata;
        this.setState({
            jiraProjectMetadata: projectMetadata,
            fetchingProjectMetadata: false,
        });

        if (projectMetadata.default_project_key) {
            this.props.onProjectChange(projectMetadata.default_project_key);
        }
    }

    handleJiraInstanceChange = (_: string, instanceID: string) => {
        if (instanceID === this.props.selectedInstanceID) {
            return;
        }

        this.props.onInstanceChange(instanceID);
        if (!this.props.hideProjectSelector) {
            this.fetchJiraProjectMetadata(instanceID);
        }
    }

    handleProjectChange = (_: string, projectID: string) => {
        this.props.onProjectChange(projectID);
    }

    render() {
        const instanceOptions: ReactSelectOption[] = this.props.connectedInstances.map((instance: Instance) => (
            {label: instance.instance_id, value: instance.instance_id}
        ));

        const label = this.state.disableInstanceSelector ? 'Instance (saved)' : 'Instance';
        let instanceSelector;
        if (this.props.connectedInstances.length > 1 && this.props.installedInstances.length > 1) {
            instanceSelector = (
                <ReactSelectSetting
                    name={'instance'}
                    label={label}
                    options={instanceOptions}
                    onChange={this.handleJiraInstanceChange}
                    value={instanceOptions.find((opt) => opt.value === this.props.selectedInstanceID)}
                    isDisabled={this.state.disableInstanceSelector}
                    required={!this.state.disableInstanceSelector}
                    theme={this.props.theme}
                    isLoading={this.state.fetchingInstances}
                />
            );
        }

        let projectSelector;
        if (this.props.selectedInstanceID && !this.props.hideProjectSelector) {
            const projectOptions = getProjectValues(this.state.jiraProjectMetadata);
            projectSelector = (
                <ReactSelectSetting
                    name={'projects'}
                    label={'Project'}
                    limitOptions={true}
                    required={true}
                    onChange={this.handleProjectChange}
                    options={projectOptions}
                    isMulti={false}
                    theme={this.props.theme}
                    value={projectOptions.find((option) => option.value === this.props.selectedProjectID) || null}
                    addValidate={this.props.addValidate}
                    removeValidate={this.props.removeValidate}
                    isLoading={this.state.fetchingProjectMetadata}
                />
            );
        }

        return (
            <React.Fragment>
                {instanceSelector}
                {projectSelector}
            </React.Fragment>
        );
    }
}
