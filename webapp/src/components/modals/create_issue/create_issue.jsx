// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import DropDown from 'components/settings/dropdown';
import FormButton from 'components/form_button';
import Input from 'components/settings/input';
import Loading from 'components/loading';

const initialState = {
    submitting:false,
    metadata: null,
    issue: {
        description: '',
        fields: [],
        project: '',
        summary: '',
        type: '',
    },
    error: null,
};

export default class CreateIssueModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        getMetadata: PropTypes.func.isRequired,
        post: PropTypes.object,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
    };

    constructor(props) {
        super(props);

        this.state = initialState;
    }

    componentWillReceiveProps(nextProps) {
        if (this.props.post !== nextProps.post && nextProps.post) {
            this.getMetadata(nextProps.post.message);
        }
    }

    getMetadata = (description) => {
        const {getMetadata} = this.props;
        const {issue} = this.state;
        getMetadata().then((meta) => {
            if (meta.error) {
                this.setState({error: meta.error.message});
                return;
            }

            console.log('create meta', meta.data);
            const updateIssue = {
                ...issue,
                description,
            };

            if (meta.data && meta.data.projects && meta.data.projects.length) {
                const pr = meta.data.projects[0];
                updateIssue.project = pr.key;
                updateIssue.type = pr.issuetypes[0].name
            }
            this.setState({
                metadata: meta.data,
                issue: updateIssue,
            });
        })
    };

    getProjectMeta = (projectKey) => {
        const {metadata} = this.state;
        return metadata.projects.find((p) => p.key === projectKey) || [];
    };

    getProjectIssueTypes = (projectKey) => {
        const project = this.getProjectMeta(projectKey);
        return project.issuetypes || [];
    };

    handleCreate = () => {
        const {create, post} = this.props;
        const {issue} = this.state;

        const data = {
            post_id: post.id,
            ...issue,
        };

        this.setState({submitting: true});

        create(data).then((created) => {
            if (created.error) {
                this.setState({error: created.error.message, submitting: false});
                return;
            }
            this.handleClose();
        })
    };

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        const {close} = this.props;
        this.setState(initialState, close);
    };

    handleDescriptionChange = (e) => {
        const description = e.target.value;
        const {issue} = this.state;
        const updateIssue = {
            ...issue,
            description,
        };

        this.setState({issue: updateIssue});
    };

    handleSettingChange = (id, value) => {
        const {issue} = this.state;
        const updateIssue = {...issue};
        switch(id) {
        case 'selectProject':
            updateIssue.project = value;
            const issueTypes = this.getProjectIssueTypes(value);
            updateIssue.type = issueTypes.length && issueTypes[0].name;
            this.setState({issue: updateIssue});
            break;
        case 'selectType':
            updateIssue.type = value;
            this.setState({issue: updateIssue});
            break;
        case 'description':
            updateIssue.description = value;
            this.setState({issue: updateIssue});
            break;
        case 'summary':
            updateIssue.summary = value;
            this.setState({issue: updateIssue});
            break;
        }
    };

    canSubmit = () => {
        const {issue, metadata, submitting} = this.state;
        const {description, project, type, summary} = issue;

        return project && type && description &&
            summary && metadata && !submitting;
    };

    render() {
        const {post, visible, theme} = this.props;
        const {issue, error, metadata, submitting} = this.state;
        const style = getStyle(theme);

        let component;
        if (error) {
            console.error('render error', error);
        }

        if (!post || !metadata) {
            component =  <Loading/>;
        } else {
            const projectsOption = (
                <DropDown
                    id='selectProject'
                    values={metadata.projects.map((p) => ({value: p.key, text: p.name}))}
                    value={issue.project}
                    label='Project'
                    required={true}
                    onChange={this.handleSettingChange}
                />
            );

            const issueTypes = (
                <DropDown
                    id='selectType'
                    values={this.getProjectIssueTypes(issue.project).map((i) => ({value: i.name, text: i.name}))}
                    value={issue.type}
                    label='Issue Type'
                    required={true}
                    onChange={this.handleSettingChange}
                />
            );

            component = (
                <div style={style.modal}>
                    {projectsOption}
                    {issueTypes}
                    <Input
                        id='summary'
                        label='Summary'
                        value={issue.summary}
                        onChange={this.handleSettingChange}
                        required={true}
                    />
                    <Input
                        id='description'
                        label='Description'
                        type='textarea'
                        value={issue.description}
                        onChange={this.handleSettingChange}
                        required={true}
                    />
                    <br/>
                </div>
            );
        }

        return (
            <Modal
                dialogClassName='modal--scroll'
                show={visible}
                onHide={this.handleClose}
                onExited={this.handleClose}
                bsSize='large'
            >
                <Modal.Header closeButton={true}>
                    <Modal.Title>
                        {'Create Jira Ticket'}
                    </Modal.Title>
                </Modal.Header>
                <form role='form'>
                    <Modal.Body ref='modalBody'>
                        {component}
                    </Modal.Body>
                    <Modal.Footer>
                        <FormButton
                            btnClass='btn-default'
                            defaultMessage='Cancel'
                            onClick={this.handleClose}
                        />
                        <FormButton
                            disabled={!this.canSubmit()}
                            btnClass='btn btn-primary'
                            saving={submitting}
                            onClick={this.handleCreate}
                        >
                            {'Create'}
                        </FormButton>
                    </Modal.Footer>
                </form>
            </Modal>
        );
    }
}

const getStyle = (theme) => ({
    modal: {
        padding: '1em',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
    descriptionArea: {
        height: 'auto',
        width: '100%',
        color: '#000'
    },
});
