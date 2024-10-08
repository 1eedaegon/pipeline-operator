/* tslint:disable */
/* eslint-disable */
/**
 * Kubernetes
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * The version of the OpenAPI document: v1.30.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

import { mapValues } from '../runtime';
/**
 * Preconditions must be fulfilled before an operation (update, delete, etc.) is carried out.
 * @export
 * @interface IoK8sApimachineryPkgApisMetaV1Preconditions
 */
export interface IoK8sApimachineryPkgApisMetaV1Preconditions {
    /**
     * Specifies the target ResourceVersion
     * @type {string}
     * @memberof IoK8sApimachineryPkgApisMetaV1Preconditions
     */
    resourceVersion?: string;
    /**
     * Specifies the target UID.
     * @type {string}
     * @memberof IoK8sApimachineryPkgApisMetaV1Preconditions
     */
    uid?: string;
}

/**
 * Check if a given object implements the IoK8sApimachineryPkgApisMetaV1Preconditions interface.
 */
export function instanceOfIoK8sApimachineryPkgApisMetaV1Preconditions(value: object): value is IoK8sApimachineryPkgApisMetaV1Preconditions {
    return true;
}

export function IoK8sApimachineryPkgApisMetaV1PreconditionsFromJSON(json: any): IoK8sApimachineryPkgApisMetaV1Preconditions {
    return IoK8sApimachineryPkgApisMetaV1PreconditionsFromJSONTyped(json, false);
}

export function IoK8sApimachineryPkgApisMetaV1PreconditionsFromJSONTyped(json: any, ignoreDiscriminator: boolean): IoK8sApimachineryPkgApisMetaV1Preconditions {
    if (json == null) {
        return json;
    }
    return {
        
        'resourceVersion': json['resourceVersion'] == null ? undefined : json['resourceVersion'],
        'uid': json['uid'] == null ? undefined : json['uid'],
    };
}

export function IoK8sApimachineryPkgApisMetaV1PreconditionsToJSON(value?: IoK8sApimachineryPkgApisMetaV1Preconditions | null): any {
    if (value == null) {
        return value;
    }
    return {
        
        'resourceVersion': value['resourceVersion'],
        'uid': value['uid'],
    };
}

